package e2e

import (
	"context"
	"errors"
	"testing"

	"bytes"
	"compress/gzip"
	"encoding/binary"
	"io"

	authzv1beta1 "cosmossdk.io/api/cosmos/authz/v1beta1"
	txv1beta1 "cosmossdk.io/api/cosmos/tx/v1beta1"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-proto/anyutil"
	gogoproto "github.com/cosmos/gogoproto/proto"
	opchildv1 "github.com/initia-labs/OPinit/api/opinit/opchild/v1"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestBatchReconstructionTest(t *testing.T) {
	l1ChainConfig := &ChainConfig{
		ChainID:        "initiation-2",
		Image:          ibc.DockerImage{Repository: "ghcr.io/initia-labs/initiad", Version: "v0.6.4", UIDGID: "1000:1000"},
		Bin:            "initiad",
		Bech32Prefix:   "init",
		Denom:          "uinit",
		Gas:            "auto",
		GasPrices:      "0.025uinit",
		GasAdjustment:  1.2,
		TrustingPeriod: "168h",
		NumValidators:  1,
		NumFullNodes:   0,
	}

	l2ChainConfig := &ChainConfig{
		ChainID:        "minimove-2",
		Image:          ibc.DockerImage{Repository: "ghcr.io/initia-labs/minimove", Version: "v0.6.5", UIDGID: "1000:1000"},
		Bin:            "minitiad",
		Bech32Prefix:   "init",
		Denom:          "umin",
		Gas:            "auto",
		GasPrices:      "0.025umin",
		GasAdjustment:  1.2,
		TrustingPeriod: "168h",
		NumValidators:  1,
		NumFullNodes:   0,
	}

	bridgeConfig := &BridgeConfig{
		SubmissionInterval:    "5s",
		FinalizationPeriod:    "10s",
		SubmissionStartHeight: "1",
		OracleEnabled:         true,
		Metadata:              "",
	}

	cases := []struct {
		name          string
		daChainConfig DAChainConfig
	}{
		{
			name: "celestia",
			daChainConfig: DAChainConfig{
				ChainConfig: ChainConfig{
					ChainID:        "celestia",
					Image:          ibc.DockerImage{Repository: "ghcr.io/celestiaorg/celestia-app", Version: "v3.2.0", UIDGID: "10001:10001"},
					Bin:            "celestia-appd",
					Bech32Prefix:   "celestia",
					Denom:          "utia",
					Gas:            "auto",
					GasPrices:      "0.25utia",
					GasAdjustment:  1.5,
					TrustingPeriod: "168h",
					NumValidators:  1,
					NumFullNodes:   0,
				},
				ChainType: ophosttypes.BatchInfo_CHAIN_TYPE_CELESTIA,
			},
		},
		{
			name: "initia",
			daChainConfig: DAChainConfig{
				ChainConfig: *l1ChainConfig,
				ChainType:   ophosttypes.BatchInfo_CHAIN_TYPE_INITIA,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			op := SetupTest(t, ctx, BotExecutor, l1ChainConfig, l2ChainConfig, &tc.daChainConfig, bridgeConfig)

			err := testutil.WaitForBlocks(ctx, 20, op.Initia, op.Minitia)
			require.NoError(t, err)

			batches, err := op.DA.QueryBatchData(ctx)
			require.NoError(t, err)

			var header executortypes.BatchDataHeader
			var chunks []executortypes.BatchDataChunk
			var blockBytes []byte

			for _, batch := range batches {
				if batch[0] == byte(executortypes.BatchDataTypeHeader) {
					header, err = executortypes.UnmarshalBatchDataHeader(batch)
					require.NoError(t, err)
				} else {
					chunk, err := executortypes.UnmarshalBatchDataChunk(batch)
					require.NoError(t, err)
					chunks = append(chunks, chunk)

					require.Equal(t, header.Start, chunk.Start)
					require.Equal(t, header.End, chunk.End)
					require.Equal(t, len(header.Checksums), int(chunk.Length))
				}

				if len(header.Checksums) == len(chunks) {
					for i, chunk := range chunks {
						require.Equal(t, i, int(chunk.Index))

						checksum := executortypes.GetChecksumFromChunk(chunk.ChunkData)
						require.Equal(t, header.Checksums[i], checksum[:])

						blockBytes = append(blockBytes, chunk.ChunkData...)
					}

					blocks, err := decompressBatch(blockBytes)
					require.NoError(t, err)

					for _, blockBz := range blocks[:len(blocks)-1] {
						block, err := unmarshalBlock(blockBz)
						require.NoError(t, err)
						require.NotNil(t, block)

						err = fillOracleData(ctx, block, op.Initia)
						require.NoError(t, err)

						pbb, err := block.ToProto()
						require.NoError(t, err)

						blockBytes, err := pbb.Marshal()
						require.NoError(t, err)

						l2Block, err := op.Minitia.GetFullNode().Client.Block(ctx, &block.Height)
						require.NoError(t, err)

						expectedBlock, err := l2Block.Block.ToProto()
						require.NoError(t, err)

						expectedBlockBytes, err := expectedBlock.Marshal()
						require.NoError(t, err)

						require.Equal(t, expectedBlockBytes, blockBytes)
					}

					chunks = make([]executortypes.BatchDataChunk, 0)
					blockBytes = make([]byte, 0)
				}
			}
		})
	}
}

func getLength(b []byte) int {
	return int(binary.LittleEndian.Uint64(b))
}

func decompressBatch(b []byte) ([][]byte, error) {
	br := bytes.NewReader(b)
	r, err := gzip.NewReader(br)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	res, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	blocksBytes := make([][]byte, 0)
	for offset := 0; offset < len(res); {
		bytesLength := getLength(res[offset : offset+8])
		offset += 8
		blocksBytes = append(blocksBytes, res[offset:offset+bytesLength])
		offset += bytesLength
	}
	return blocksBytes, nil
}

// unmarshal block without validation.
//
// the validation will be performed after oracle data is fetched.
func unmarshalBlock(blockBz []byte) (*cmtypes.Block, error) {
	pbb := new(cmtproto.Block)
	err := gogoproto.Unmarshal(blockBz, pbb)
	if err != nil {
		return nil, err
	}

	return BlockFromProtoWithNoValidation(pbb)
}

// BlockFromProtoWithNoValidation sets a protobuf Block to the given pointer.
func BlockFromProtoWithNoValidation(bp *cmtproto.Block) (*cmtypes.Block, error) {
	if bp == nil {
		return nil, errors.New("nil block")
	}

	b := new(cmtypes.Block)
	h, err := cmtypes.HeaderFromProto(&bp.Header)
	if err != nil {
		return nil, err
	}
	b.Header = h
	data, err := cmtypes.DataFromProto(&bp.Data)
	if err != nil {
		return nil, err
	}
	b.Data = data
	if err := b.Evidence.FromProto(&bp.Evidence); err != nil {
		return nil, err
	}

	if bp.LastCommit != nil {
		lc, err := cmtypes.CommitFromProto(bp.LastCommit)
		if err != nil {
			return nil, err
		}
		b.LastCommit = lc
	}

	return b, nil
}

func fillOracleData(ctx context.Context, block *cmtypes.Block, chain *L1Chain) error {
	for i, txBytes := range block.Txs {
		var raw txv1beta1.TxRaw
		if err := proto.Unmarshal(txBytes, &raw); err != nil {
			return err
		}

		var body txv1beta1.TxBody
		if err := proto.Unmarshal(raw.BodyBytes, &body); err != nil {
			return err
		}

		for _, anyMsg := range body.Messages {
			switch anyMsg.TypeUrl {
			case "/opinit.opchild.v1.MsgUpdateOracle":
				msg := new(opchildv1.MsgUpdateOracle)
				err := anyMsg.UnmarshalTo(msg)
				if err != nil {
					return err
				}
				height := int64(msg.Height)
				originBlock, err := chain.GetFullNode().Client.Block(ctx, &height)
				if err != nil {
					return err
				}
				msg.Data = originBlock.Block.Txs[0]

				err = anyutil.MarshalFrom(anyMsg, msg, proto.MarshalOptions{})
				if err != nil {
					return errors.Join(errors.New("failed to marshal oracle msg"), err)
				}
			case "/cosmos.authz.v1beta1.MsgExec":
				authzMsg := new(authzv1beta1.MsgExec)
				err := anyMsg.UnmarshalTo(authzMsg)
				if err != nil {
					return err
				}
				if len(authzMsg.Msgs) != 0 || authzMsg.Msgs[0].TypeUrl != "/opinit.opchild.v1.MsgUpdateOracle" {
					continue
				}
				msg := new(opchildv1.MsgUpdateOracle)

				err = authzMsg.Msgs[0].UnmarshalTo(msg)
				if err != nil {
					return err
				}
				height := int64(msg.Height)
				originBlock, err := chain.GetFullNode().Client.Block(ctx, &height)
				if err != nil {
					return err
				}
				msg.Data = originBlock.Block.Txs[0]

				err = anyutil.MarshalFrom(authzMsg.Msgs[0], msg, proto.MarshalOptions{})
				if err != nil {
					return errors.Join(errors.New("failed to marshal oracle msg"), err)
				}

				err = anyutil.MarshalFrom(anyMsg, authzMsg, proto.MarshalOptions{})
				if err != nil {
					return errors.Join(errors.New("failed to marshal oracle msg"), err)
				}
			default:
				continue
			}
		}

		bodyBytes, err := proto.Marshal(&body)
		if err != nil {
			return err
		}
		raw.BodyBytes = bodyBytes
		convertedTxBytes, err := proto.Marshal(&raw)
		if err != nil {
			return err
		}
		block.Txs[i] = convertedTxBytes
	}
	return nil
}
