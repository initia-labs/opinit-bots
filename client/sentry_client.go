package http

import (
	"context"

	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	cmtypes "github.com/cometbft/cometbft/types"

	"github.com/initia-labs/opinit-bots/sentry_integration"
	"github.com/initia-labs/opinit-bots/types"
)

// SentryHTTP is a wrapper around HTTP that adds Sentry tracing functionality.
type SentryHTTP struct {
	*HTTP
}

// NewSentryHTTP wraps the original HTTP client with Sentry tracing capabilities.
func NewSentryHTTP(remote, wsEndpoint string) (*SentryHTTP, error) {
	client, err := New(remote, wsEndpoint)
	if err != nil {
		return nil, err
	}

	return &SentryHTTP{
		HTTP: client,
	}, nil
}

func (c *SentryHTTP) Block(ctx types.Context, height *int64) (*ctypes.ResultBlock, error) {
	span, ctx := sentry_integration.StartSentrySpan(ctx, "SentryHTTPBlock", "Calling /block from RPCs")
	defer span.Finish()
	return c.baseRPCClient.Block(ctx.Context(), height)
}

func (c *SentryHTTP) BlockResults(ctx types.Context, height *int64) (*ctypes.ResultBlockResults, error) {
	span, ctx := sentry_integration.StartSentrySpan(ctx, "SentryHTTPBlockResults", "Calling /block_results from RPCs")
	defer span.Finish()
	return c.baseRPCClient.BlockResults(ctx.Context(), height)
}

func (c *SentryHTTP) Status(ctx types.Context) (*ctypes.ResultStatus, error) {
	span, ctx := sentry_integration.StartSentrySpan(ctx, "SentryHTTPStatus", "Calling /status from RPCs")
	defer span.Finish()
	return c.baseRPCClient.Status(ctx.Context())
}

func (c *SentryHTTP) Header(ctx types.Context, height *int64) (*ctypes.ResultHeader, error) {
	span, ctx := sentry_integration.StartSentrySpan(ctx, "SentryHTTPHeader", "Calling /header from RPCs")
	defer span.Finish()
	return c.baseRPCClient.Header(ctx.Context(), height)
}

func (c *SentryHTTP) BroadcastTxSync(
	ctx types.Context,
	tx cmtypes.Tx,
) (*ctypes.ResultBroadcastTx, error) {
	span, ctx := sentry_integration.StartSentrySpan(ctx, "SentryBroadcastTxSync", "broadcast_tx_sync")
	defer span.Finish()
	return c.BroadcastTxSyncWithBaseContext(ctx.Context(), tx)
}

func (c *SentryHTTP) BroadcastTxSyncWithBaseContext(
	ctx context.Context,
	tx cmtypes.Tx,
) (*ctypes.ResultBroadcastTx, error) {
	return c.baseRPCClient.BroadcastTxSync(ctx, tx)
}
