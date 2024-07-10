package executor

import (
	"context"

	bottypes "github.com/initia-labs/opinit-bots-go/bot/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
	"go.uber.org/zap"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
)

var _ bottypes.Bot = &Executor{}

type Executor struct {
	host  *host
	child *child

	cfg    *executortypes.Config
	db     types.DB
	logger *zap.Logger
}

func NewExecutor(cfg *executortypes.Config, db types.DB, logger *zap.Logger, cdc codec.Codec, txConfig client.TxConfig) *Executor {
	executor := &Executor{
		cfg:    cfg,
		db:     db,
		logger: logger,
	}

	executor.host = NewHost(executor, cfg.BridgeId, cfg.HostNode, db.WithPrefix([]byte(nodetypes.HostNodeName)), logger.Named(nodetypes.HostNodeName), cdc, txConfig)
	executor.child = NewChild(executor, cfg.BridgeId, cfg.ChildNode, db.WithPrefix([]byte(nodetypes.ChildNodeName)), logger.Named(nodetypes.ChildNodeName), cdc, txConfig)

	executor.host.registerChildNode(executor.child)
	executor.child.registerHostNode(executor.host)

	return executor
}

func (ex Executor) Start(cmdCtx context.Context) error {
	hostCtx, hostDone := context.WithCancel(cmdCtx)
	childCtx, childDone := context.WithCancel(cmdCtx)
	ex.host.Start(hostCtx)
	ex.child.Start(childCtx)

	<-cmdCtx.Done()

	// TODO: safely shut down
	hostDone()
	childDone()

	return ex.db.Close()
}
