package e2e

import (
	"context"
	"errors"
	"testing"

	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"io"

	authzv1beta1 "cosmossdk.io/api/cosmos/authz/v1beta1"
	txv1beta1 "cosmossdk.io/api/cosmos/tx/v1beta1"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-proto/anyutil"
	gogoproto "github.com/cosmos/gogoproto/proto"
	ibcclienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibctmlightclients "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	opchildv1 "github.com/initia-labs/OPinit/api/opinit/opchild/v1"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestBatchReconstruction(t *testing.T) {
	l1ChainConfig := &ChainConfig{
		ChainID:        "initiation-2",
		Image:          ibc.DockerImage{Repository: "ghcr.io/initia-labs/initiad", Version: "v1.0.0-beta.8", UIDGID: "1000:1000"},
		Bin:            "initiad",
		Bech32Prefix:   "init",
		Denom:          "uinit",
		Gas:            "auto",
		GasPrices:      "0.025uinit",
		GasAdjustment:  1.2,
		TrustingPeriod: "1h",
		NumValidators:  1,
		NumFullNodes:   0,
	}

	l2ChainConfig := &ChainConfig{
		ChainID:        "minimove-2",
		Image:          ibc.DockerImage{Repository: "ghcr.io/initia-labs/minimove", Version: "v1.0.0-beta.13", UIDGID: "1000:1000"},
		Bin:            "minitiad",
		Bech32Prefix:   "init",
		Denom:          "umin",
		Gas:            "auto",
		GasPrices:      "0.025umin",
		GasAdjustment:  1.2,
		TrustingPeriod: "1h",
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
		relayerImpl   ibc.RelayerImplementation
	}{
		{
			name: "initia with go relayer",
			daChainConfig: DAChainConfig{
				ChainConfig: *l1ChainConfig,
				ChainType:   ophosttypes.BatchInfo_INITIA,
			},
			relayerImpl: ibc.CosmosRly,
		},
		{
			name: "initia with hermes relayer",
			daChainConfig: DAChainConfig{
				ChainConfig: *l1ChainConfig,
				ChainType:   ophosttypes.BatchInfo_INITIA,
			},
			relayerImpl: ibc.Hermes,
		},
		{
			name: "celestia",
			daChainConfig: DAChainConfig{
				ChainConfig: ChainConfig{
					ChainID:        "celestia",
					Image:          ibc.DockerImage{Repository: "ghcr.io/celestiaorg/celestia-app", Version: "v3.3.1", UIDGID: "10001:10001"},
					Bin:            "celestia-appd",
					Bech32Prefix:   "celestia",
					Denom:          "utia",
					Gas:            "auto",
					GasPrices:      "0.25utia",
					GasAdjustment:  1.5,
					TrustingPeriod: "1h",
					NumValidators:  1,
					NumFullNodes:   0,
				},
				ChainType: ophosttypes.BatchInfo_CELESTIA,
			},
			relayerImpl: ibc.CosmosRly,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			op := SetupTest(t, ctx, BotExecutor, l1ChainConfig, l2ChainConfig, &tc.daChainConfig, bridgeConfig, tc.relayerImpl)

			err := testutil.WaitForBlocks(ctx, 20, op.Initia, op.Minitia)
			require.NoError(t, err)

			genesisRes, err := op.Minitia.GetFullNode().Client.Genesis(ctx)
			require.NoError(t, err)

			batches, err := op.DA.QueryBatchData(ctx)
			require.NoError(t, err)

			genesisChecker := false

			var header executortypes.BatchDataHeader
			var chunks []executortypes.BatchDataChunk
			var blockBytes []byte
			var genesisChunks []executortypes.BatchDataGenesis
			var genesisBz []byte

			for _, batch := range batches {
				switch executortypes.BatchDataType(batch[0]) {
				case executortypes.BatchDataTypeHeader:
					header, err = executortypes.UnmarshalBatchDataHeader(batch)
					require.NoError(t, err)
				case executortypes.BatchDataTypeChunk:
					chunk, err := executortypes.UnmarshalBatchDataChunk(batch)
					require.NoError(t, err)
					chunks = append(chunks, chunk)

					require.Equal(t, header.Start, chunk.Start)
					require.Equal(t, header.End, chunk.End)
					require.Equal(t, len(header.Checksums), int(chunk.Length))
				case executortypes.BatchDataTypeGenesis:
					if genesisChecker {
						require.Fail(t, "genesis already found")
					}

					genesisChunk, err := executortypes.UnmarshalBatchDataGenesis(batch)
					require.NoError(t, err)
					genesisChunks = append(genesisChunks, genesisChunk)

					require.Equal(t, len(genesisChunks)-1, int(genesisChunk.Index))
					genesisBz = append(genesisBz, genesisChunk.ChunkData...)
				default:
					require.Fail(t, "unknown batch data type")
				}

				if !genesisChecker && len(genesisChunks) == int(genesisChunks[0].Length) {
					expectedBz, err := json.Marshal(genesisRes.Genesis)
					require.NoError(t, err)
					require.Equal(t, expectedBz, genesisBz)
					genesisChecker = true
				}

				if genesisChecker && len(header.Checksums) > 0 && len(header.Checksums) == len(chunks) {
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

						err = fillData(ctx, block, op.Initia)
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

func fillData(ctx context.Context, block *cmtypes.Block, chain *L1Chain) error {
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
				if len(authzMsg.Msgs) != 1 || authzMsg.Msgs[0].TypeUrl != "/opinit.opchild.v1.MsgUpdateOracle" {
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
			case "/ibc.core.client.v1.MsgUpdateClient":
				updateClientMsg := new(ibcclienttypes.MsgUpdateClient)
				err := updateClientMsg.Unmarshal(anyMsg.Value)
				if err != nil {
					return err
				}

				if updateClientMsg.ClientMessage.TypeUrl != "/ibc.lightclients.tendermint.v1.Header" {
					continue
				}

				tmHeader := new(ibctmlightclients.Header)
				err = tmHeader.Unmarshal(updateClientMsg.ClientMessage.Value)
				if err != nil {
					return err
				}

				// fill ValidatorSet
				height := tmHeader.SignedHeader.Commit.Height
				validators, err := getAllValidators(ctx, chain, height)
				if err != nil {
					return err
				}
				cmtValidators, _, err := toCmtProtoValidators(validators)
				if err != nil {
					return err
				}
				if tmHeader.ValidatorSet == nil {
					tmHeader.ValidatorSet = new(cmtproto.ValidatorSet)
				}
				tmHeader.ValidatorSet.Validators = cmtValidators
				for _, val := range cmtValidators {
					if bytes.Equal(val.Address, tmHeader.SignedHeader.Header.ProposerAddress) {
						tmHeader.ValidatorSet.Proposer = val
					}
				}

				// fill TrustedValidators
				height = int64(tmHeader.TrustedHeight.RevisionHeight)
				validators, err = getAllValidators(ctx, chain, height)
				if err != nil {
					return err
				}
				cmtValidators, _, err = toCmtProtoValidators(validators)
				if err != nil {
					return err
				}
				blockHeader, err := chain.GetFullNode().Client.Header(ctx, &height)
				if err != nil {
					return err
				}
				if tmHeader.TrustedValidators == nil {
					tmHeader.TrustedValidators = new(cmtproto.ValidatorSet)
				}
				tmHeader.TrustedValidators.Validators = cmtValidators
				for _, val := range cmtValidators {
					if bytes.Equal(val.Address, blockHeader.Header.ProposerAddress.Bytes()) {
						tmHeader.TrustedValidators.Proposer = val
					}
				}

				// fill commit signatures
				height = tmHeader.SignedHeader.Commit.Height + 1
				block, err := chain.GetFullNode().Client.Block(ctx, &height)
				if err != nil {
					return err
				}

				for sigIndex, signature := range tmHeader.SignedHeader.Commit.Signatures {
					if len(signature.Signature) == 2 {
						// fill signature
						blockSigIndex := int(signature.Signature[0]) + int(signature.Signature[1])<<8
						tmHeader.SignedHeader.Commit.Signatures[sigIndex] = *block.Block.LastCommit.Signatures[blockSigIndex].ToProto()
					}
				}

				updateClientMsg.ClientMessage.Value, err = tmHeader.Marshal()
				if err != nil {
					return errors.Join(errors.New("failed to marshal tm header"), err)
				}

				anyMsg.Value, err = updateClientMsg.Marshal()
				if err != nil {
					return errors.Join(errors.New("failed to marshal update client msg"), err)
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

func getAllValidators(ctx context.Context, chain *L1Chain, height int64) ([]*cmtypes.Validator, error) {
	page := 1
	perPage := 100

	validators := make([]*cmtypes.Validator, 0)
	for {
		result, err := chain.GetFullNode().Client.Validators(ctx, &height, &page, &perPage)
		if err != nil {
			return nil, err
		}
		validators = append(validators, result.Validators...)
		page++
		if len(validators) >= result.Total {
			break
		}
	}
	return validators, nil
}

func toCmtProtoValidators(validators []*cmtypes.Validator) ([]*cmtproto.Validator, int64, error) {
	protoValidators := make([]*cmtproto.Validator, 0, len(validators))
	totalVotingPower := int64(0)

	for i := range validators {
		protoValidator, err := validators[i].ToProto()
		if err != nil {
			return nil, 0, err
		}
		protoValidator.ProposerPriority = 0
		totalVotingPower += protoValidator.VotingPower

		protoValidators = append(protoValidators, protoValidator)
	}
	return protoValidators, totalVotingPower, nil
}
