package types

import (
	"context"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Context struct {
	baseCtx context.Context

	logger *zap.Logger

	errGrp          *errgroup.Group
	pollingInterval time.Duration
	txTimeout       time.Duration
	rpcTimeout      time.Duration
	homePath        string
}

func NewContext(baseCtx context.Context, logger *zap.Logger, homePath string) Context {
	return Context{
		baseCtx: baseCtx,

		logger:   logger,
		homePath: homePath,
	}
}

var _ context.Context = (*Context)(nil)

func (c Context) Value(key any) any {
	return c.baseCtx.Value(key)
}

func (c Context) Deadline() (deadline time.Time, ok bool) {
	return c.baseCtx.Deadline()
}

func (c Context) Done() <-chan struct{} {
	return c.baseCtx.Done()
}

func (c Context) Err() error {
	return c.baseCtx.Err()
}

func (c Context) WithContext(ctx context.Context) Context {
	c.baseCtx = ctx
	return c
}

func (c Context) WithLogger(logger *zap.Logger) Context {
	c.logger = logger
	return c
}

func (c Context) WithErrGrp(errGrp *errgroup.Group) Context {
	c.errGrp = errGrp
	return c
}

func (c Context) WithPollingInterval(interval time.Duration) Context {
	c.pollingInterval = interval
	return c
}

func (c Context) WithTxTimeout(timeout time.Duration) Context {
	c.txTimeout = timeout
	return c
}

func (c Context) WithRPCTimeout(timeout time.Duration) Context {
	c.rpcTimeout = timeout
	return c
}

func (c Context) WithHomePath(homePath string) Context {
	c.homePath = homePath
	return c
}

func (c Context) Context() context.Context {
	return c.baseCtx
}

func (c Context) Logger() *zap.Logger {
	return c.logger
}

func (c Context) ErrGrp() *errgroup.Group {
	return c.errGrp
}

func (c Context) PollingInterval() time.Duration {
	return c.pollingInterval
}

func (c Context) TxTimeout() time.Duration {
	return c.txTimeout
}

func (c Context) RPCTimeout() time.Duration {
	return c.rpcTimeout
}

func (c Context) HomePath() string {
	return c.homePath
}
