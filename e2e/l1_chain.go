package e2e

import (
	"context"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
)

type L1ChainNode struct {
	*cosmos.ChainNode
}

func NewL1ChainNode(chainNode *cosmos.ChainNode) *L1ChainNode {
	return &L1ChainNode{
		ChainNode: chainNode,
	}
}

type L1Chain struct {
	*cosmos.CosmosChain

	OutputSubmitter ibc.Wallet
	BatchSubmitter  ibc.Wallet
	Challenger      ibc.Wallet
}

func NewL1Chain(cosmosChain *cosmos.CosmosChain, outputSubmitter ibc.Wallet, batchSubmitter ibc.Wallet, challenger ibc.Wallet) *L1Chain {
	return &L1Chain{
		CosmosChain:     cosmosChain,
		OutputSubmitter: outputSubmitter,
		BatchSubmitter:  batchSubmitter,
		Challenger:      challenger,
	}
}

func (l1 *L1Chain) GetNode() *L1ChainNode {
	return NewL1ChainNode(l1.CosmosChain.GetNode())
}

func (l1 *L1Chain) GetFullNode() *L1ChainNode {
	return NewL1ChainNode(l1.CosmosChain.GetFullNode())
}

func (l1 *L1Chain) CreateBridge(ctx context.Context, keyName string, configPath string) (string, error) {
	return l1.GetFullNode().ExecTx(ctx, keyName, "ophost", "create-bridge", configPath)
}

func (l1 *L1Chain) QueryBridge(ctx context.Context, bridgeId uint64) (*ophosttypes.QueryBridgeResponse, error) {
	return ophosttypes.NewQueryClient(l1.GetFullNode().GrpcConn).Bridge(ctx, &ophosttypes.QueryBridgeRequest{BridgeId: bridgeId})
}
