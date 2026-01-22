package batchsubmitter

import (
	"bytes"
	"slices"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"

	cmttypes "github.com/cometbft/cometbft/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibctmlightclients "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	"github.com/initia-labs/opinit-bots/txutils"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
)

// emptyOracleData converts the MsgUpdateOracle messages's data field to empty
// to decrease the size of the batch.
func (bs *BatchSubmitter) emptyOracleData(pbb *cmtproto.Block) (*cmtproto.Block, error) {
	txs := pbb.Data.GetTxs()
	if len(txs) == 0 {
		return pbb, nil
	}
	txBytes := txs[0]

	txConfig := bs.node.GetTxConfig()
	tx, err := txutils.DecodeTx(txConfig, txBytes)
	if err != nil {
		// ignore not registered tx in codec
		return pbb, nil
	}

	msgs := tx.GetMsgs()
	// oracle tx has only one message
	if len(msgs) != 1 {
		return pbb, nil
	}

	switch msg := msgs[0].(type) {
	case *opchildtypes.MsgUpdateOracle:
		msg.Data = []byte{}
	case *authz.MsgExec:
		if len(msg.Msgs) != 1 || msg.Msgs[0].TypeUrl != "/opinit.opchild.v1.MsgUpdateOracle" {
			return pbb, nil
		}
		oracleMsg := &opchildtypes.MsgUpdateOracle{}
		err = bs.node.Codec().UnpackAny(msg.Msgs[0], &oracleMsg)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unpack oracle msg from authz msg")
		}
		oracleMsg.Data = []byte{}
		msgs[0], err = childprovider.CreateAuthzMsg(msg.Grantee, oracleMsg)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create authz msg")
		}
	}

	tx, err = txutils.ChangeMsgsFromTx(txConfig, tx, []sdk.Msg{msgs[0]})
	if err != nil {
		return nil, errors.Wrap(err, "failed to change msgs from tx")
	}
	convertedTxBytes, err := txutils.EncodeTx(txConfig, tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode tx")
	}
	pbb.Data.Txs[0] = convertedTxBytes

	return pbb, nil
}

// emptyRelayOracleData converts the MsgRelayOracleData message's prices field to empty
// to decrease the size of the batch. The oracle data can be restored from L1 using the
// oracle price hash and proof.
func (bs *BatchSubmitter) emptyRelayOracleData(pbb *cmtproto.Block) (*cmtproto.Block, error) {
	txs := pbb.Data.GetTxs()
	if len(txs) == 0 {
		return pbb, nil
	}
	txBytes := txs[0]

	txConfig := bs.node.GetTxConfig()
	tx, err := txutils.DecodeTx(txConfig, txBytes)
	if err != nil {
		// ignore not registered tx in codec
		return pbb, nil
	}

	msgs := tx.GetMsgs()
	// relay oracle tx has only one message
	if len(msgs) != 1 {
		return pbb, nil
	}

	switch msg := msgs[0].(type) {
	case *opchildtypes.MsgRelayOracleData:
		// empty prices array and proof to reduce batch size.
		// keeping only oracle_price_hash, l1_block_height, l1_block_time, and proof_height.
		// these fields are sufficient to restore both the prices and proof from L1.
		msg.OracleData.Prices = []opchildtypes.OraclePriceData{}
		msg.OracleData.Proof = []byte{}
	case *authz.MsgExec:
		if len(msg.Msgs) != 1 || msg.Msgs[0].TypeUrl != "/opinit.opchild.v1.MsgRelayOracleData" {
			return pbb, nil
		}
		relayMsg := &opchildtypes.MsgRelayOracleData{}
		err = bs.node.Codec().UnpackAny(msg.Msgs[0], &relayMsg)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unpack relay oracle msg from authz msg")
		}
		relayMsg.OracleData.Prices = []opchildtypes.OraclePriceData{}
		relayMsg.OracleData.Proof = []byte{}
		msgs[0], err = childprovider.CreateAuthzMsg(msg.Grantee, relayMsg)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create authz msg")
		}
	default:
		return pbb, nil
	}

	tx, err = txutils.ChangeMsgsFromTx(txConfig, tx, []sdk.Msg{msgs[0]})
	if err != nil {
		return nil, errors.Wrap(err, "failed to change msgs from tx")
	}
	convertedTxBytes, err := txutils.EncodeTx(txConfig, tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode tx")
	}
	pbb.Data.Txs[0] = convertedTxBytes

	return pbb, nil
}

// emptyUpdateClientData converts the MsgUpdateClient messages's validator set and part of signature fields to empty
func (bs *BatchSubmitter) emptyUpdateClientData(ctx types.Context, pbb *cmtproto.Block) (*cmtproto.Block, error) {
	blockQuerier := func(height int64) (*coretypes.ResultBlock, error) {
		ticker := time.NewTicker(ctx.PollingInterval())
		defer ticker.Stop()

		var block *coretypes.ResultBlock
		var err error
		for retry := 1; retry <= types.MaxRetryCount; retry++ {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-ticker.C:
			}
			block, err = bs.host.QueryBlock(ctx, height)
			if err != nil {
				continue
			}
		}
		return block, err
	}

	for txIndex, txBytes := range pbb.Data.GetTxs() {
		txConfig := bs.node.GetTxConfig()
		tx, err := txutils.DecodeTx(txConfig, txBytes)
		if err != nil {
			// ignore not registered tx in codec
			continue
		}

		msgs := tx.GetMsgs()
		for msgIndex, msg := range msgs {
			switch msg := msg.(type) {
			case *ibcclienttypes.MsgUpdateClient:
				if msg.ClientMessage.TypeUrl != "/ibc.lightclients.tendermint.v1.Header" {
					continue
				}

				clientMsg := &ibctmlightclients.Header{}
				err = bs.node.Codec().UnpackAny(msg.ClientMessage, &clientMsg)
				if err != nil {
					return nil, errors.Wrap(err, "failed to unpack oracle msg from authz msg")
				}

				if clientMsg.Header.ChainID != bs.host.ChainId() {
					continue
				}

				block, err := blockQuerier(clientMsg.Header.Height + 1)
				if err != nil {
					return nil, errors.Wrap(err, "failed to query block")
				}

				for sigIndex, signature := range clientMsg.Commit.Signatures {
					if blockSigIndex := slices.IndexFunc(block.Block.LastCommit.Signatures, func(blockSig cmttypes.CommitSig) bool {
						if signature.ValidatorAddress != nil &&
							bytes.Equal(blockSig.ValidatorAddress.Bytes(), signature.ValidatorAddress) &&
							blockSig.Timestamp.Equal(signature.Timestamp) &&
							bytes.Equal(blockSig.Signature, signature.Signature) &&
							uint8(blockSig.BlockIDFlag) == uint8(signature.BlockIdFlag) {
							return true
						}
						return false
					}); blockSigIndex != -1 {
						// assume that the length of the validator set is less than 65536
						if blockSigIndex >= 1<<16 {
							return nil, errors.New("validator set length is greater than 65536")
						}

						newSig := []byte{}
						newSig = append(newSig, byte(blockSigIndex%(1<<8)))
						newSig = append(newSig, byte(blockSigIndex>>8))

						clientMsg.SignedHeader.Commit.Signatures[sigIndex].Signature = newSig
						clientMsg.SignedHeader.Commit.Signatures[sigIndex].ValidatorAddress = []byte{}
						clientMsg.SignedHeader.Commit.Signatures[sigIndex].Timestamp = time.Time{}
						clientMsg.SignedHeader.Commit.Signatures[sigIndex].BlockIdFlag = 0
					}
				}

				// empty validator set and trusted validators
				clientMsg.ValidatorSet = &cmtproto.ValidatorSet{
					TotalVotingPower: clientMsg.ValidatorSet.TotalVotingPower,
				}
				clientMsg.TrustedValidators = &cmtproto.ValidatorSet{
					TotalVotingPower: clientMsg.TrustedValidators.TotalVotingPower,
				}

				msgs[msgIndex], err = ibcclienttypes.NewMsgUpdateClient(msg.ClientId, clientMsg, msg.Signer)
				if err != nil {
					return nil, errors.Wrap(err, "failed to create new msg update client")
				}
			}
		}
		tx, err = txutils.ChangeMsgsFromTx(txConfig, tx, msgs)
		if err != nil {
			return nil, errors.Wrap(err, "failed to change msgs from tx")
		}
		convertedTxBytes, err := txutils.EncodeTx(txConfig, tx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to encode tx")
		}
		pbb.Data.Txs[txIndex] = convertedTxBytes
	}
	return pbb, nil
}
