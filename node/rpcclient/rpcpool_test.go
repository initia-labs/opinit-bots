package rpcclient

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"

	"github.com/initia-labs/opinit-bots/types"
)

// createTestContext creates a test context with default RPC timeout
func createTestContext(logger *zap.Logger) types.Context {
	return types.NewContext(context.Background(), logger, "/tmp").
		WithRPCTimeout(5 * time.Second)
}

func TestRPCPool_GetCurrentEndpoint(t *testing.T) {
	logger := zaptest.NewLogger(t)
	endpoints := []string{"doi", "moro", "rene"}
	pool := NewRPCPool(createTestContext(logger), endpoints, logger)

	// Initial endpoint should be the first one
	assert.Equal(t, "doi", pool.GetCurrentEndpoint())
}

func TestRPCPool_MoveToNextEndpoint(t *testing.T) {
	logger := zaptest.NewLogger(t)
	endpoints := []string{"doi", "moro", "rene"}
	pool := NewRPCPool(createTestContext(logger), endpoints, logger)

	// Move to next endpoint
	assert.Equal(t, "moro", pool.MoveToNextEndpoint())
	assert.Equal(t, "moro", pool.GetCurrentEndpoint())

	// Move to next endpoint again
	assert.Equal(t, "rene", pool.MoveToNextEndpoint())
	assert.Equal(t, "rene", pool.GetCurrentEndpoint())

	// Move to next endpoint should wrap around
	assert.Equal(t, "doi", pool.MoveToNextEndpoint())
	assert.Equal(t, "doi", pool.GetCurrentEndpoint())
}

func TestRPCPool_ExecuteWithFallback_Success(t *testing.T) {
	logger := zaptest.NewLogger(t)
	endpoints := []string{"doi", "moro", "rene"}
	pool := NewRPCPool(createTestContext(logger), endpoints, logger)

	// Function succeeds on first try
	callCount := 0
	err := pool.ExecuteWithFallback(context.Background(), func(ctx context.Context) error {
		callCount++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
	assert.Equal(t, "doi", pool.GetCurrentEndpoint())
}

func TestRPCPool_ExecuteWithFallback_FallbackSuccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	endpoints := []string{"doi", "moro", "rene"}
	pool := NewRPCPool(createTestContext(logger), endpoints, logger)

	// Function fails on first endpoint, succeeds on second
	callCount := 0
	err := pool.ExecuteWithFallback(context.Background(), func(ctx context.Context) error {
		callCount++
		if callCount == 1 {
			return errors.New("first endpoint failed")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, callCount)
	assert.Equal(t, "moro", pool.GetCurrentEndpoint())
}

func TestRPCPool_ExecuteWithFallback_AllFail(t *testing.T) {
	logger := zaptest.NewLogger(t)
	endpoints := []string{"doi", "moro"}
	pool := NewRPCPool(createTestContext(logger), endpoints, logger)
	pool.maxRetries = 1 // Set to 1 for faster test

	// All endpoints fail
	callCount := 0
	err := pool.ExecuteWithFallback(context.Background(), func(ctx context.Context) error {
		callCount++
		return errors.New("endpoint failed")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "all RPC endpoints failed after 1 retries")
	// 2 endpoints + 2 more for 1 retry = 4 calls
	assert.Equal(t, 4, callCount)
}

func TestRPCPool_ExecuteWithFallback_Timeout(t *testing.T) {
	logger := zaptest.NewLogger(t)
	endpoints := []string{"doi"}
	pool := NewRPCPool(createTestContext(logger), endpoints, logger)
	pool.rpcTimeout = 100 * time.Millisecond

	// Function takes too long
	err := pool.ExecuteWithFallback(context.Background(), func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return nil
		}
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestRPCPool_ExecuteWithFallback_RetrySuccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	endpoints := []string{"doi"}
	pool := NewRPCPool(createTestContext(logger), endpoints, logger)
	pool.maxRetries = 2
	pool.retryInterval = 10 * time.Millisecond

	// Function fails on first try, succeeds on retry
	callCount := 0
	err := pool.ExecuteWithFallback(context.Background(), func(ctx context.Context) error {
		callCount++
		if callCount <= 1 {
			return errors.New("first try failed")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestRPCPool_ExecuteWithFallback_ContextCancellation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	endpoints := []string{"doi", "moro"}
	pool := NewRPCPool(createTestContext(logger), endpoints, logger)

	// Create a context that will be canceled
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// Function should return context canceled error
	err := pool.ExecuteWithFallback(ctx, func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return nil
		}
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestRPCPool_Logging(t *testing.T) {
	// Create a logger that captures logs
	core, recorded := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	endpoints := []string{"doi", "moro"}
	pool := NewRPCPool(createTestContext(logger), endpoints, logger)

	// Function fails on first endpoint, succeeds on second
	_ = pool.ExecuteWithFallback(context.Background(), func(ctx context.Context) error {
		if pool.GetCurrentEndpoint() == "doi" {
			return errors.New("first endpoint failed")
		}
		return nil
	})

	// Check that the failure was logged
	logs := recorded.All()
	assert.True(t, len(logs) > 0)

	foundFailureLog := false
	for _, log := range logs {
		if log.Message == "RPC request failed, trying next endpoint" {
			foundFailureLog = true
			break
		}
	}
	assert.True(t, foundFailureLog, "Should have logged endpoint failure")
}
