package e2e

import (
	"context"
	"fmt"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
)

type L2ChainNode struct {
	*cosmos.ChainNode
}

func NewL2ChainNode(chainNode *cosmos.ChainNode) *L2ChainNode {
	return &L2ChainNode{
		ChainNode: chainNode,
	}
}

type L2Chain struct {
	*cosmos.CosmosChain
	BridgeExecutor ibc.Wallet
	Validator      ibc.Wallet
}

func NewL2Chain(cosmosChain *cosmos.CosmosChain, bridgeExecutor ibc.Wallet, validator ibc.Wallet) *L2Chain {
	return &L2Chain{
		CosmosChain:    cosmosChain,
		BridgeExecutor: bridgeExecutor,
		Validator:      validator,
	}
}

func (l2 *L2Chain) GetNode() *L2ChainNode {
	return NewL2ChainNode(l2.CosmosChain.GetNode())
}

func (l2 *L2Chain) GetFullNode() *L2ChainNode {
	return NewL2ChainNode(l2.CosmosChain.GetFullNode())
}

func (l2 *L2Chain) SetBridgeInfo(ctx context.Context, bridgeId uint64, bridgeAddr string, l1ChainId string, l1ClientId string, configPath string) (string, error) {
	cmds := []string{
		"opchild", "set-bridge-info", fmt.Sprintf("%d", bridgeId), bridgeAddr, l1ChainId, l1ClientId, configPath,
		"--gas-prices", "0umin",
	}
	return l2.GetFullNode().ExecTx(ctx, l2.BridgeExecutor.KeyName(), cmds...)
}
