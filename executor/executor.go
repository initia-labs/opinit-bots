package executor

import (
	"context"
	"fmt"

	"cosmossdk.io/core/address"
	bottypes "github.com/initia-labs/opinit-bots-go/bot/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
	"go.uber.org/zap"

	"github.com/initia-labs/opinit-bots-go/node"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
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

	hostProcessedMsgs           []nodetypes.ProcessedMsgs
	childMsgQueue               []sdk.Msg
	childAllPendingTxsProcessed chan struct{}
}

func NewExecutor(cfg *executortypes.Config, db types.DB, logger *zap.Logger, cdc codec.Codec, txConfig client.TxConfig) *Executor {
	hostNode, err := node.NewNode(nodetypes.HostNodeName, cfg.HostNode, db.WithPrefix([]byte(nodetypes.HostNodeName)), logger.Named(nodetypes.HostNodeName), cdc, txConfig)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	childNode, err := node.NewNode(nodetypes.ChildNodeName, cfg.ChildNode, db.WithPrefix([]byte(nodetypes.ChildNodeName)), logger.Named(nodetypes.ChildNodeName), cdc, txConfig)
	if err != nil {
		fmt.Println(err)
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

		hostProcessedMsgs:           make([]nodetypes.ProcessedMsgs, 0),
		childMsgQueue:               make([]sdk.Msg, 0),
		childAllPendingTxsProcessed: make(chan struct{}),
	}

	executor.registerHostHandlers()
	executor.registerChildHandlers()

	return executor
}

func (ex Executor) Start(cmdCtx context.Context) error {
	hostCtx, hostDone := context.WithCancel(cmdCtx)
	childCtx, childDone := context.WithCancel(cmdCtx)
	ex.hostNode.Start(hostCtx)
	ex.childNode.Start(childCtx)

	<-cmdCtx.Done()

	// TODO: safely shut down
	hostDone()
	childDone()

	return ex.db.Close()
}

func (ex Executor) registerHostHandlers() {
	ex.hostNode.RegisterBeginBlockHandler(ex.hostBeginBlockHandler)
	ex.hostNode.RegisterTxHandler(ex.hostTxHandler)
	ex.hostNode.RegisterEventHandler(ophosttypes.EventTypeInitiateTokenDeposit, ex.initiateDepositHandler)
	ex.hostNode.RegisterEndBlockHandler(ex.hostEndBlockHandler)
}

func (ex Executor) registerChildHandlers() {
	ex.childNode.RegisterEventHandler(opchildtypes.EventTypeFinalizeTokenDeposit, ex.finalizeDepositHandler)
	ex.childNode.RegisterEventHandler(opchildtypes.EventTypeUpdateOracle, ex.updateOracleHandler)
	ex.childNode.RegisterEndBlockHandler(ex.childEndBlockHandler)
}
