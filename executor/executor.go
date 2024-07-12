package executor

import (
	"context"

	"github.com/initia-labs/opinit-bots-go/executor/child"
	"github.com/initia-labs/opinit-bots-go/executor/host"

	bottypes "github.com/initia-labs/opinit-bots-go/bot/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	"github.com/initia-labs/opinit-bots-go/types"
	"go.uber.org/zap"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
)

var _ bottypes.Bot = &Executor{}

type Executor struct {
	host  *host.Host
	child *child.Child

	cfg    *executortypes.Config
	db     types.DB
	logger *zap.Logger
}

func NewExecutor(cfg *executortypes.Config, db types.DB, logger *zap.Logger, cdc codec.Codec, txConfig client.TxConfig) *Executor {
	h := &host.Host{}
	ch := &child.Child{}

	executor := &Executor{
		host:  h,
		child: ch,

		cfg:    cfg,
		db:     db,
		logger: logger,
	}

	*h = *host.NewHost(cfg.BridgeId, cfg.HostNode, db.WithPrefix([]byte(executortypes.HostNodeName)), logger.Named(executortypes.HostNodeName), cdc, txConfig, ch)
	*ch = *child.NewChild(cfg.BridgeId, cfg.ChildNode, db.WithPrefix([]byte(executortypes.ChildNodeName)), logger.Named(executortypes.ChildNodeName), cdc, txConfig, h)
	return executor
}

func (ex *Executor) Start(cmdCtx context.Context) error {
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
