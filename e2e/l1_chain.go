package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"time"

	"github.com/avast/retry-go/v4"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"go.uber.org/zap"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type L1ChainNode struct {
	*cosmos.ChainNode
	log *zap.Logger
}

func NewL1ChainNode(log *zap.Logger, chainNode *cosmos.ChainNode) *L1ChainNode {
	return &L1ChainNode{
		ChainNode: chainNode,
		log:       log,
	}
}

type L1Chain struct {
	*cosmos.CosmosChain

	OutputSubmitter ibc.Wallet
	Challenger      ibc.Wallet

	log *zap.Logger
}

func NewL1Chain(log *zap.Logger, cosmosChain *cosmos.CosmosChain, outputSubmitter ibc.Wallet, challenger ibc.Wallet) *L1Chain {
	return &L1Chain{
		log:             log,
		CosmosChain:     cosmosChain,
		OutputSubmitter: outputSubmitter,
		Challenger:      challenger,
	}
}

func (l1 *L1Chain) GetNode() *L1ChainNode {
	return NewL1ChainNode(l1.log, l1.CosmosChain.GetNode())
}

func (l1 *L1Chain) GetFullNode() *L1ChainNode {
	return NewL1ChainNode(l1.log, l1.CosmosChain.GetFullNode())
}

func (l1 *L1Chain) WaitUntilOutputIsFinalized(ctx context.Context, bridgeId uint64, outputId uint64) error {
	var lastFinalizedOutput *ophosttypes.QueryLastFinalizedOutputResponse
	var bridgeResponse *ophosttypes.QueryBridgeResponse
	var err error

	if err := retry.Do(func() error {
		bridgeResponse, err = l1.QueryBridge(ctx, bridgeId)
		return err
	}, retry.Context(ctx), RtyAtt, RtyDel, RtyErr); err != nil {
		return err
	}

	delay := 2 * time.Second
	maxAttempts := (bridgeResponse.BridgeConfig.SubmissionInterval + bridgeResponse.BridgeConfig.FinalizationPeriod) / delay

	if err := retry.Do(func() error {
		lastFinalizedOutput, err = l1.QueryLastFinalizedOutput(ctx, bridgeId)
		if err != nil {
			l1.log.Error("querying last finalized output failed", zap.Error(err))
			return err
		} else if lastFinalizedOutput.OutputIndex < outputId {
			return fmt.Errorf("output %d is not finalized yet", outputId)
		}
		return nil
	}, retry.Context(ctx), retry.Attempts(uint(maxAttempts)), retry.Delay(delay), RtyErr); err != nil { //nolint
		return err
	}
	return nil
}

// Tx

func (l1 *L1Chain) CreateBridge(ctx context.Context, keyName string, configPath string) (string, error) {
	return l1.GetFullNode().ExecTx(ctx, keyName, "ophost", "create-bridge", configPath)
}

func (l1 *L1Chain) InitiateTokenDeposit(ctx context.Context, keyName string, bridgeId uint64, to, amount, data string, onlySend bool) (string, error) {
	node := l1.GetFullNode()
	commands := []string{"ophost", "initiate-token-deposit", fmt.Sprintf("%d", bridgeId), to, amount, data}
	if onlySend {
		stdout, _, err := node.Exec(ctx, l1.GetFullNode().TxCommand(keyName, commands...), node.Chain.Config().Env)
		return string(stdout), err
	}
	return node.ExecTx(ctx, keyName, commands...)
}

func (l1 *L1Chain) FinalizeTokenWithdrawal(ctx context.Context, keyName string, bridgeId uint64, outputIndex uint64, sequence uint64, withdrawalProofs [][]byte, from, to string, amount sdk.Coin, version []byte, storageRoot []byte, lastBlockHash []byte, onlySend bool) (string, error) {
	node := l1.GetFullNode()

	withdrawal := ophosttypes.NewMsgFinalizeTokenWithdrawal(
		"", bridgeId, outputIndex, sequence, withdrawalProofs, from, to, amount, version, storageRoot, lastBlockHash,
	)

	withdrawalBz, err := json.Marshal(withdrawal)
	if err != nil {
		return "", err
	}
	err = node.WriteFile(ctx, withdrawalBz, "withdrawal-info.json")
	if err != nil {
		return "", err
	}

	commands := []string{"ophost", "finalize-token-withdrawal", path.Join(node.HomeDir(), "withdrawal-info.json")}
	if onlySend {
		stdout, _, err := node.Exec(ctx, l1.GetFullNode().TxCommand(keyName, commands...), node.Chain.Config().Env)
		return string(stdout), err
	}
	return node.ExecTx(ctx, keyName, commands...)
}

// Query

func (l1 *L1Chain) QueryBridge(ctx context.Context, bridgeId uint64) (*ophosttypes.QueryBridgeResponse, error) {
	return ophosttypes.NewQueryClient(l1.GetFullNode().GrpcConn).Bridge(ctx, &ophosttypes.QueryBridgeRequest{BridgeId: bridgeId})
}

func (l1 *L1Chain) QueryTokenPairByL1Denom(ctx context.Context, bridgeId uint64, denom string) (*ophosttypes.QueryTokenPairByL1DenomResponse, error) {
	return ophosttypes.NewQueryClient(l1.GetFullNode().GrpcConn).TokenPairByL1Denom(ctx, &ophosttypes.QueryTokenPairByL1DenomRequest{BridgeId: bridgeId, L1Denom: denom})
}

func (l1 *L1Chain) QueryOutputProposal(ctx context.Context, bridgeId uint64, outputId uint64) (*ophosttypes.QueryOutputProposalResponse, error) {
	return ophosttypes.NewQueryClient(l1.GetFullNode().GrpcConn).OutputProposal(ctx, &ophosttypes.QueryOutputProposalRequest{BridgeId: bridgeId, OutputIndex: outputId})
}

func (l1 *L1Chain) QueryLastFinalizedOutput(ctx context.Context, bridgeId uint64) (*ophosttypes.QueryLastFinalizedOutputResponse, error) {
	return ophosttypes.NewQueryClient(l1.GetFullNode().GrpcConn).LastFinalizedOutput(ctx, &ophosttypes.QueryLastFinalizedOutputRequest{BridgeId: bridgeId})
}
