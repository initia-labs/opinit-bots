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
	pool, err := NewRPCPool(createTestContext(logger), endpoints, logger)
	assert.NoError(t, err)

	// Initial endpoint should be the first one
	assert.Equal(t, "doi", pool.GetCurrentClient().endpoint)
}

func TestRPCPool_MoveToNextHealthyClient(t *testing.T) {
	logger := zaptest.NewLogger(t)
	endpoints := []string{"doi", "moro", "rene"}
	pool, err := NewRPCPool(createTestContext(logger), endpoints, logger)
	assert.NoError(t, err)

	// Move to next healthy client
	client := pool.MoveToNextHealthyClient()
	assert.NotNil(t, client)
	assert.Equal(t, "moro", client.endpoint)
	assert.Equal(t, "moro", pool.GetCurrentClient().endpoint)

	// Move to next healthy client again
	client = pool.MoveToNextHealthyClient()
	assert.NotNil(t, client)
	assert.Equal(t, "rene", client.endpoint)
	assert.Equal(t, "rene", pool.GetCurrentClient().endpoint)

	// Move to next healthy client should wrap around
	client = pool.MoveToNextHealthyClient()
	assert.NotNil(t, client)
	assert.Equal(t, "doi", client.endpoint)
	assert.Equal(t, "doi", pool.GetCurrentClient().endpoint)
}

func TestRPCPool_ExecuteWithFallback_Success(t *testing.T) {
	logger := zaptest.NewLogger(t)
	endpoints := []string{"doi", "moro", "rene"}
	pool, err := NewRPCPool(createTestContext(logger), endpoints, logger)
	assert.NoError(t, err)

	// Function succeeds on first try
	callCount := 0
	err = pool.ExecuteWithFallback(context.Background(), func(ctx context.Context) error {
		callCount++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
	assert.Equal(t, "doi", pool.GetCurrentClient().endpoint)
}

func TestRPCPool_ExecuteWithFallback_FallbackSuccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	endpoints := []string{"doi", "moro", "rene"}
	pool, err := NewRPCPool(createTestContext(logger), endpoints, logger)
	assert.NoError(t, err)

	// Function fails on first endpoint, succeeds on second
	callCount := 0
	err = pool.ExecuteWithFallback(context.Background(), func(ctx context.Context) error {
		callCount++
		if callCount == 1 {
			return errors.New("first endpoint failed")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, callCount)
	assert.Equal(t, "moro", pool.GetCurrentClient().endpoint)
}

func TestRPCPool_ExecuteWithFallback_AllFail(t *testing.T) {
	logger := zaptest.NewLogger(t)
	endpoints := []string{"doi", "moro"}
	pool, err := NewRPCPool(createTestContext(logger), endpoints, logger)
	assert.NoError(t, err)
	pool.maxRetries = 1 // Set to 1 for faster test

	// All endpoints fail
	callCount := 0
	err = pool.ExecuteWithFallback(context.Background(), func(ctx context.Context) error {
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
	pool, err := NewRPCPool(createTestContext(logger), endpoints, logger)
	assert.NoError(t, err)
	pool.rpcTimeout = 100 * time.Millisecond

	// Function takes too long
	err = pool.ExecuteWithFallback(context.Background(), func(ctx context.Context) error {
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
	pool, err := NewRPCPool(createTestContext(logger), endpoints, logger)
	assert.NoError(t, err)
	pool.maxRetries = 2
	pool.retryInterval = 10 * time.Millisecond

	// Function fails on first try, succeeds on retry
	callCount := 0
	err = pool.ExecuteWithFallback(context.Background(), func(ctx context.Context) error {
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
	pool, err := NewRPCPool(createTestContext(logger), endpoints, logger)
	assert.NoError(t, err)

	// Create a context that will be canceled
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// Function should return context canceled error
	err = pool.ExecuteWithFallback(ctx, func(ctx context.Context) error {
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
	pool, err := NewRPCPool(createTestContext(logger), endpoints, logger)
	assert.NoError(t, err)

	// Function fails on first endpoint, succeeds on second
	_ = pool.ExecuteWithFallback(context.Background(), func(ctx context.Context) error {
		if pool.GetCurrentClient().endpoint == "doi" {
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

// TestNewRPCPool_ValidEndpoints tests NewRPCPool with valid endpoints
func TestNewRPCPool_ValidEndpoints(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := createTestContext(logger)

	validEndpoints := []string{"http://localhost:26657", "http://localhost:26658"}
	pool, err := NewRPCPool(ctx, validEndpoints, logger)

	assert.NoError(t, err, "NewRPCPool should succeed with valid endpoints")
	assert.NotNil(t, pool, "Pool should not be nil")
	assert.Equal(t, len(validEndpoints), len(pool.clients), "Pool should have all valid endpoints")
}

// TestNewRPCPool_InvalidEndpoints tests NewRPCPool with invalid endpoints
func TestNewRPCPool_InvalidEndpoints(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := createTestContext(logger)

	invalidEndpoints := []string{"://malformed-url", "://another-malformed-url"}
	pool, err := NewRPCPool(ctx, invalidEndpoints, logger)

	assert.Error(t, err, "NewRPCPool should fail with invalid endpoints")
	assert.Nil(t, pool, "Pool should be nil when all endpoints are invalid")
	assert.Contains(t, err.Error(), "no valid endpoints found", "Error should indicate no valid endpoints")
}

// TestNewRPCPool_MixedEndpoints tests NewRPCPool with mixed valid and invalid endpoints
func TestNewRPCPool_MixedEndpoints(t *testing.T) {
	// Create a logger with observer to capture warning logs
	core, recorded := observer.New(zap.WarnLevel)
	logger := zap.New(core)
	ctx := createTestContext(logger)

	mixedEndpoints := []string{"http://localhost:26657", "://malformed-url", "http://localhost:26658"}
	pool, err := NewRPCPool(ctx, mixedEndpoints, logger)

	assert.NoError(t, err, "NewRPCPool should succeed with mixed endpoints (invalid ones filtered out)")
	assert.NotNil(t, pool, "Pool should not be nil")
	assert.Equal(t, 2, len(pool.clients), "Pool should have only the valid endpoints")

	// Check that invalid endpoints were logged as warnings
	logs := recorded.All()
	foundWarning := false
	for _, log := range logs {
		if log.Message == "Removing invalid endpoint from pool" {
			foundWarning = true
			break
		}
	}
	assert.True(t, foundWarning, "Should have logged warning for invalid endpoint")
}

// TestNewRPCPool_EmptyEndpoints tests NewRPCPool with empty endpoints (should return error)
func TestNewRPCPool_EmptyEndpoints(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := createTestContext(logger)

	emptyEndpoints := []string{}
	pool, err := NewRPCPool(ctx, emptyEndpoints, logger)
	
	assert.Error(t, err, "NewRPCPool should return error with empty endpoints")
	assert.Nil(t, pool, "Pool should be nil when no endpoints provided")
	assert.Contains(t, err.Error(), "no RPC endpoints provided", "Error should indicate no endpoints provided")
}
