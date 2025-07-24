package rpcclient

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/cosmos/cosmos-sdk/codec"
	clienthttp "github.com/initia-labs/opinit-bots/client"
	"github.com/initia-labs/opinit-bots/types"
)

const (
	// DefaultRPCTimeout is the default timeout for RPC requests in seconds
	DefaultRPCTimeout = 5
)

// RPCPool manages multiple RPC endpoints with fallback and retry logic
type RPCPool struct {
	endpoints     []string
	currentIndex  int
	mu            sync.RWMutex
	rpcTimeout    time.Duration
	logger        *zap.Logger
	maxRetries    int
	retryInterval time.Duration
}

// NewRPCPool creates a new RPC pool with the given endpoints
func NewRPCPool(endpoints []string, logger *zap.Logger) *RPCPool {
	if len(endpoints) == 0 {
		panic("endpoints slice cannot be empty")
	}

	// Get timeout from environment variable or use default
	timeoutStr := os.Getenv("RPC_TIMEOUT_SECONDS")
	timeout := DefaultRPCTimeout
	if timeoutStr != "" {
		if t, err := strconv.Atoi(timeoutStr); err == nil && t > 0 {
			timeout = t
		} else {
			logger.Warn("Invalid RPC_TIMEOUT_SECONDS value, using default",
				zap.String("value", timeoutStr),
				zap.Int("default", DefaultRPCTimeout))
		}
	}

	return &RPCPool{
		endpoints:     endpoints,
		currentIndex:  0,
		mu:            sync.RWMutex{},
		rpcTimeout:    time.Duration(timeout) * time.Second,
		logger:        logger,
		maxRetries:    types.MaxRetryCount,
		retryInterval: 1 * time.Second,
	}
}

// GetCurrentEndpoint returns the current RPC endpoint
func (p *RPCPool) GetCurrentEndpoint() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.endpoints[p.currentIndex]
}

// MoveToNextEndpoint moves to the next RPC endpoint
func (p *RPCPool) MoveToNextEndpoint() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentIndex = (p.currentIndex + 1) % len(p.endpoints)
	endpoint := p.endpoints[p.currentIndex]
	p.logger.Info("Switching to next RPC endpoint", zap.String("endpoint", endpoint))
	return endpoint
}

// getCurrentIndex returns the current index (thread-safe)
func (p *RPCPool) getCurrentIndex() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.currentIndex
}

// setCurrentIndex sets the current index (thread-safe)
func (p *RPCPool) setCurrentIndex(index int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentIndex = index
}

// tryAllEndpoints tries the given function on all endpoints once
// Returns nil on first success or the last encountered error if all endpoints fail
func (p *RPCPool) tryAllEndpoints(ctx context.Context, fn func(context.Context) error, retryAttempt int) error {
	// Try current endpoint first (which should be the last successful one)
	currentEndpoint := p.GetCurrentEndpoint()
	startIndex := p.getCurrentIndex()

	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, p.rpcTimeout)

	if retryAttempt == 0 {
		p.logger.Debug("Trying RPC endpoint", zap.String("endpoint", currentEndpoint))
	} else {
		p.logger.Debug("Retrying RPC endpoint",
			zap.String("endpoint", currentEndpoint),
			zap.Int("retry", retryAttempt))
	}

	err := fn(timeoutCtx)
	cancel()
	if err == nil {
		// Current endpoint worked, no need to try others
		return nil
	}

	var lastErr = err
	if retryAttempt == 0 {
		p.logger.Warn("RPC request failed, trying next endpoint",
			zap.String("endpoint", currentEndpoint),
			zap.String("error", err.Error()))
	} else {
		p.logger.Warn("RPC request failed during retry, trying next endpoint",
			zap.String("endpoint", currentEndpoint),
			zap.String("error", err.Error()),
			zap.Int("retry", retryAttempt))
	}

	// Current endpoint failed, try the remaining endpoints
	for i := 1; i < len(p.endpoints); i++ {
		p.MoveToNextEndpoint()
		currentEndpoint = p.GetCurrentEndpoint()

		// Create a timeout context
		timeoutCtx, cancel := context.WithTimeout(ctx, p.rpcTimeout)

		if retryAttempt == 0 {
			p.logger.Debug("Trying RPC endpoint", zap.String("endpoint", currentEndpoint))
		} else {
			p.logger.Debug("Retrying RPC endpoint",
				zap.String("endpoint", currentEndpoint),
				zap.Int("retry", retryAttempt))
		}

		err := fn(timeoutCtx)
		cancel()
		if err == nil {
			// This endpoint worked, keep it as current for future requests
			return nil
		}

		lastErr = err
		if retryAttempt == 0 {
			p.logger.Warn("RPC request failed, trying next endpoint",
				zap.String("endpoint", currentEndpoint),
				zap.String("error", err.Error()))
		} else {
			p.logger.Warn("RPC request failed during retry, trying next endpoint",
				zap.String("endpoint", currentEndpoint),
				zap.String("error", err.Error()),
				zap.Int("retry", retryAttempt))
		}
	}

	// Reset to original position if all endpoints failed
	if retryAttempt == 0 {
		p.setCurrentIndex(startIndex)
	}

	return lastErr
}

// ExecuteWithFallback executes the given function with fallback to other endpoints if it fails
// and retries with exponential backoff if all endpoints fail
func (p *RPCPool) ExecuteWithFallback(ctx context.Context, fn func(context.Context) error) error {
	// First attempt: try all endpoints once
	err := p.tryAllEndpoints(ctx, fn, 0)
	if err == nil {
		return nil
	}

	// If all endpoints failed, retry with exponential backoff
	var lastErr error
	for retry := 0; retry < p.maxRetries; retry++ {
		// Calculate backoff duration
		backoffDuration := time.Duration(math.Pow(2, float64(retry))) * p.retryInterval

		p.logger.Info("All RPC endpoints failed, retrying after backoff",
			zap.Duration("backoff", backoffDuration),
			zap.Int("retry", retry+1),
			zap.Int("max_retries", p.maxRetries))

		// Wait for backoff duration
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoffDuration):
		}

		// Try all endpoints again
		err := p.tryAllEndpoints(ctx, fn, retry+1)
		if err == nil {
			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("all RPC endpoints failed after %d retries: %w", p.maxRetries, lastErr)
}

// CreateRPCClient creates a new RPC client with the given codec and RPC addresses
func CreateRPCClient(cdc codec.Codec, rpcAddresses []string, logger *zap.Logger) (*RPCClient, error) {
	if len(rpcAddresses) == 0 {
		return nil, errors.New("no RPC addresses provided")
	}

	// Create RPC pool
	pool := NewRPCPool(rpcAddresses, logger)

	// Create HTTP client with the first endpoint
	client, err := clienthttp.New(pool.GetCurrentEndpoint(), "/websocket")
	if err != nil {
		return nil, err
	}

	// Create RPC client
	rpcClient := &RPCClient{
		HTTP: client,
		cdc:  cdc,
		pool: pool,
	}

	return rpcClient, nil
}
