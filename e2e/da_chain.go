package e2e

import (
	"context"
	"fmt"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"go.uber.org/zap"

	txv1beta1 "cosmossdk.io/api/cosmos/tx/v1beta1"
	comettypes "github.com/cometbft/cometbft/types"
	"google.golang.org/protobuf/proto"

	ophostv1 "github.com/initia-labs/OPinit/api/opinit/ophost/v1"

	opcelestia "github.com/initia-labs/opinit-bots/types/celestia"
)

type DAChainNode struct {
	*cosmos.ChainNode
	log *zap.Logger
}

func NewDAChainNode(log *zap.Logger, chainNode *cosmos.ChainNode) *DAChainNode {
	return &DAChainNode{
		ChainNode: chainNode,
		log:       log,
	}
}

type DAChain struct {
	*cosmos.CosmosChain

	ChainType      ophosttypes.BatchInfo_ChainType
	BatchSubmitter ibc.Wallet

	log *zap.Logger
}

func NewDAChain(log *zap.Logger, cosmosChain *cosmos.CosmosChain, chainType ophosttypes.BatchInfo_ChainType, batchSubmitter ibc.Wallet) *DAChain {
	return &DAChain{
		log:            log,
		CosmosChain:    cosmosChain,
		ChainType:      chainType,
		BatchSubmitter: batchSubmitter,
	}
}

func (da *DAChain) GetNode() *DAChainNode {
	return NewDAChainNode(da.log, da.CosmosChain.GetNode())
}

func (da *DAChain) GetFullNode() *DAChainNode {
	return NewDAChainNode(da.log, da.CosmosChain.GetFullNode())
}

func (da *DAChain) QueryBatchData(ctx context.Context) ([][]byte, error) {
	switch da.ChainType {
	case ophosttypes.BatchInfo_CHAIN_TYPE_INITIA:
		return da.QueryInitiaBatchData(ctx)
	case ophosttypes.BatchInfo_CHAIN_TYPE_CELESTIA:
		return da.QueryCelestiaBatchData(ctx)
	}
	return nil, fmt.Errorf("unsupported chain type")
}

func (da *DAChain) QueryInitiaBatchData(ctx context.Context) ([][]byte, error) {
	if da.ChainType != ophosttypes.BatchInfo_CHAIN_TYPE_INITIA {
		return nil, fmt.Errorf("unmatched chain type")
	}

	data := [][]byte{}
	page := 1
	perPage := 100

	for {
		txsResult, err := da.GetFullNode().Client.TxSearch(ctx, "message.action='/opinit.ophost.v1.MsgRecordBatch'", false, &page, &perPage, "asc")
		if err != nil {
			return nil, err
		}

		for _, tx := range txsResult.Txs {
			var raw txv1beta1.TxRaw
			if err := proto.Unmarshal(tx.Tx, &raw); err != nil {
				return nil, err
			}

			var body txv1beta1.TxBody
			if err := proto.Unmarshal(raw.BodyBytes, &body); err != nil {
				return nil, err
			}

			if len(body.Messages) == 0 {
				continue
			}

			recordBatch := new(ophostv1.MsgRecordBatch)
			if err := body.Messages[0].UnmarshalTo(recordBatch); err != nil {
				return nil, err
			}
			data = append(data, recordBatch.BatchBytes)
		}

		if txsResult.TotalCount <= page*100 {
			break
		}
	}
	return data, nil
}

func (da *DAChain) QueryCelestiaBatchData(ctx context.Context) ([][]byte, error) {
	if da.ChainType != ophosttypes.BatchInfo_CHAIN_TYPE_CELESTIA {
		return nil, fmt.Errorf("unmatched chain type")
	}

	data := [][]byte{}
	page := 1
	perPage := 100

	var lastBlock *comettypes.Block

	for {
		txsResult, err := da.GetFullNode().Client.TxSearch(ctx, "message.action='/celestia.blob.v1.MsgPayForBlobs'", false, &page, &perPage, "asc")
		if err != nil {
			return nil, err
		}

		for _, tx := range txsResult.Txs {
			if lastBlock == nil || tx.Height != lastBlock.Height {
				blockResult, err := da.GetFullNode().Client.Block(ctx, &tx.Height)
				if err != nil {
					return nil, err
				}
				lastBlock = blockResult.Block
			}

			var blobTx opcelestia.BlobTx
			err = blobTx.Unmarshal(lastBlock.Txs[tx.Index])
			if err != nil {
				return nil, err
			}
			data = append(data, blobTx.Blobs[0].Data)
		}

		if txsResult.TotalCount <= page*100 {
			break
		}
	}
	return data, nil
}
