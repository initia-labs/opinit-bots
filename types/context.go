package types

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"
)

type contextKey string

var (
	ContextKeyErrGrp          = contextKey("ErrGrp")
	ContextKeyPollingInterval = contextKey("PollingInterval")
	ContextKeyTxTimeout       = contextKey("TxTimeout")
)

func WithErrGrp(ctx context.Context, errGrp *errgroup.Group) context.Context {
	return context.WithValue(ctx, ContextKeyErrGrp, errGrp)
}

func ErrGrp(ctx context.Context) *errgroup.Group {
	return ctx.Value(ContextKeyErrGrp).(*errgroup.Group)
}

func WithPollingInterval(ctx context.Context, interval time.Duration) context.Context {
	return context.WithValue(ctx, ContextKeyPollingInterval, interval)
}

func PollingInterval(ctx context.Context) time.Duration {
	interval := ctx.Value(ContextKeyPollingInterval)
	if interval == nil {
		return 100 * time.Millisecond
	}
	return ctx.Value(ContextKeyPollingInterval).(time.Duration)
}
