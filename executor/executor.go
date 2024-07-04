package executor

import (
	"context"

	"cosmossdk.io/core/address"
	bottypes "github.com/initia-labs/opinit-bots-go/bot/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	"github.com/initia-labs/opinit-bots-go/types"
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

	cfg    *executortypes.Config
	db     types.DB
	logger *zap.Logger

	cdc codec.Codec
	ac  address.Codec
}

func NewExecutor(cfg *executortypes.Config, db types.DB, logger *zap.Logger, cdc codec.Codec, txConfig client.TxConfig) *Executor {
	hostNode, err := node.NewNode(types.HostNodeName, cfg.HostNode, db.WithPrefix(types.HostNodePrefix), logger, cdc, txConfig)
	if err != nil {
		panic(err)
	}

	childNode, err := node.NewNode(types.ChildNodeName, cfg.ChildNode, db.WithPrefix(types.ChildNodePrefix), logger, cdc, txConfig)
	if err != nil {
		panic(err)
	}

	executor := &Executor{
		hostNode:  hostNode,
		childNode: childNode,

		cfg:    cfg,
		db:     db,
		logger: logger,

		cdc: cdc,
		ac:  cdc.InterfaceRegistry().SigningContext().AddressCodec(),
	}

	executor.registerHostHandlers()
	executor.registerChildHandlers()

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

	return ex.db.Close()
}

func (ex Executor) registerHostHandlers() {
	ex.hostNode.RegisterTxHandler(ex.hostTxHandler)

	ex.hostNode.RegisterEventHandler(ophosttypes.EventTypeInitiateTokenDeposit, ex.initiateDepositHandler)
}

func (ex Executor) registerChildHandlers() {
	ex.childNode.RegisterEventHandler(opchildtypes.EventTypeFinalizeTokenDeposit, ex.finalizeDepositHandler)
	ex.childNode.RegisterEventHandler(opchildtypes.EventTypeUpdateOracle, ex.updateOracleHandler)
}
