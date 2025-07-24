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

	clienthttp "github.com/initia-labs/opinit-bots/client"
	"github.com/cosmos/cosmos-sdk/codec"
)

const (
	// DefaultRPCTimeout is the default timeout for RPC requests in seconds
	DefaultRPCTimeout = 5
	// DefaultMaxRetries is the default maximum number of retries for RPC requests
	DefaultMaxRetries = 3
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
		maxRetries:    DefaultMaxRetries,
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

// ExecuteWithFallback executes the given function with fallback to other endpoints if it fails
func (p *RPCPool) ExecuteWithFallback(ctx context.Context, fn func(context.Context) error) error {
	// Try all endpoints
	for i := 0; i < len(p.endpoints); i++ {
		currentEndpoint := p.GetCurrentEndpoint()
		
		// Create a timeout context
		timeoutCtx, cancel := context.WithTimeout(ctx, p.rpcTimeout)
		defer cancel()
		
		p.logger.Debug("Trying RPC endpoint", zap.String("endpoint", currentEndpoint))
		
		err := fn(timeoutCtx)
		if err == nil {
			return nil
		}
		
		p.logger.Warn("RPC request failed, trying next endpoint", 
			zap.String("endpoint", currentEndpoint), 
			zap.String("error", err.Error()))
		
		// Move to the next endpoint
		p.MoveToNextEndpoint()
	}
	
	// If all endpoints failed, retry with exponential backoff
	return p.retryWithBackoff(ctx, fn)
}

// retryWithBackoff retries the given function with exponential backoff
func (p *RPCPool) retryWithBackoff(ctx context.Context, fn func(context.Context) error) error {
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
		for i := 0; i < len(p.endpoints); i++ {
			currentEndpoint := p.GetCurrentEndpoint()
			
			// Create a timeout context
			timeoutCtx, cancel := context.WithTimeout(ctx, p.rpcTimeout)
			defer cancel()
			
			p.logger.Debug("Retrying RPC endpoint", 
				zap.String("endpoint", currentEndpoint), 
				zap.Int("retry", retry+1))
			
			err := fn(timeoutCtx)
			if err == nil {
				return nil
			}
			
			lastErr = err
			p.logger.Warn("RPC request failed during retry, trying next endpoint", 
				zap.String("endpoint", currentEndpoint), 
				zap.String("error", err.Error()), 
				zap.Int("retry", retry+1))
			
			// Move to the next endpoint
			p.MoveToNextEndpoint()
		}
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