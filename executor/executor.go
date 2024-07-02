package executor

import (
	"context"

	"cosmossdk.io/core/address"
	bottypes "github.com/initia-labs/opinit-bots-go/bot/types"
	"github.com/initia-labs/opinit-bots-go/executor/types"
	"go.uber.org/zap"

	"github.com/initia-labs/opinit-bots-go/node"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
)

var _ bottypes.Bot = &Executor{}

type Executor struct {
	hostNode  *node.Node
	childNode *node.Node

	cfg    *types.Config
	logger *zap.Logger

	cdc codec.Codec
	ac  address.Codec
}

func NewExecutor(cfg *types.Config, logger *zap.Logger, cdc codec.Codec, txConfig client.TxConfig) *Executor {
	hostNode, err := node.NewNode("host", cfg.HostNode, logger, cdc, txConfig)
	if err != nil {
		panic(err)
	}

	childNode, err := node.NewNode("child", cfg.ChildNode, logger, cdc, txConfig)
	if err != nil {
		panic(err)
	}

	executor := &Executor{
		hostNode:  hostNode,
		childNode: childNode,

		cfg:    cfg,
		logger: logger,

		cdc: cdc,
		ac:  cdc.InterfaceRegistry().SigningContext().AddressCodec(),
	}

	executor.registerHostEventHandlers()
	executor.registerChildEventHandlers()

	return executor
}

func (ex Executor) Start(cmdCtx context.Context) error {
	hostCtx, hostDone := context.WithCancel(cmdCtx)
	go ex.hostNode.BlockProcessLooper(hostCtx)
	go ex.hostNode.TxBroadCastLooper(hostCtx)
	childCtx, childDone := context.WithCancel(cmdCtx)
	go ex.childNode.BlockProcessLooper(childCtx)
	go ex.childNode.TxBroadCastLooper(childCtx)

	<-cmdCtx.Done()
	hostDone()
	childDone()
	return nil
}

func (ex Executor) registerHostEventHandlers() {
	ex.hostNode.RegisterEventHandler(ophosttypes.EventTypeInitiateTokenDeposit, ex.initiateDepositHandler)
}

func (ex Executor) registerChildEventHandlers() {
	ex.childNode.RegisterEventHandler(opchildtypes.EventTypeFinalizeTokenDeposit, ex.finalizeDepositHandler)
}
