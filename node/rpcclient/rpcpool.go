package rpcclient

import (
	"context"
	"fmt"
	"sort"
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

	// Scoring system constants
	DefaultInitialScore    = 100.0           // Initial score for new endpoints
	ScoreDecayOnFailure    = 10.0            // Points to subtract on failure
	ScoreDecayOnTimeout    = 20.0            // Points to subtract on timeout (more severe)
	ScoreIncreaseOnSuccess = 5.0             // Points to add on success
	MinScore               = 1.0             // Minimum score to prevent complete exclusion
	MaxScore               = 200.0           // Maximum score to prevent unbounded growth
	ScoreResetInterval     = 5 * time.Minute // How often to reset scores
)

// RPCClientInfo holds information about an RPC client and its health status
type RPCClientInfo struct {
	client       *clienthttp.HTTP
	endpoint     string
	healthy      bool
	lastError    error
	lastCheck    time.Time
	score        float64   // Endpoint score for prioritization (higher is better)
	successCount int64     // Number of successful requests
	failureCount int64     // Number of failed requests
	timeoutCount int64     // Number of timeout failures
	lastReset    time.Time // Last time scores were reset
}

// RPCPool manages multiple RPC endpoints with persistent HTTP clients, fallback and retry logic
type RPCPool struct {
	clients        []*RPCClientInfo
	currentIndex   int
	mu             sync.RWMutex
	rpcTimeout     time.Duration
	logger         *zap.Logger
	maxRetries     int
	retryInterval  time.Duration
	lastScoreReset time.Time // Last time scores were reset across all endpoints
}

// NewRPCPool creates a new RPC pool with the given endpoints.
// Invalid endpoints (those that fail clienthttp.New) are automatically dropped during initialization
// and logged as warnings. Returns an error if all provided endpoints are invalid and no valid
// endpoints remain after filtering.
func NewRPCPool(ctx types.Context, endpoints []string, logger *zap.Logger) (*RPCPool, error) {
	if len(endpoints) == 0 {
		return nil, errors.New("no RPC endpoints provided")
	}

	// Get timeout from context or use default
	rpcTimeout := ctx.RPCTimeout()
	if rpcTimeout == 0 {
		rpcTimeout = time.Duration(DefaultRPCTimeout) * time.Second
	}

	// Create HTTP clients for each endpoint, filtering out invalid ones
	var clients []*RPCClientInfo
	now := time.Now()

	for _, endpoint := range endpoints {
		client, err := clienthttp.New(endpoint, "/websocket")
		if err != nil {
			// Log the error and remove invalid endpoint from the pool
			logger.Warn("Removing invalid endpoint from pool",
				zap.String("endpoint", endpoint),
				zap.Error(err))
			continue // Skip this endpoint entirely
		}

		// Only add valid endpoints and their clients to the pool
		clients = append(clients, &RPCClientInfo{
			client:       client,
			endpoint:     endpoint,
			healthy:      true,
			lastError:    nil,
			lastCheck:    now,
			score:        DefaultInitialScore,
			successCount: 0,
			failureCount: 0,
			timeoutCount: 0,
			lastReset:    now,
		})
	}

	// Ensure we have at least one valid endpoint
	if len(clients) == 0 {
		return nil, errors.New("no valid endpoints found - all endpoints failed to create HTTP clients")
	}

	return &RPCPool{
		clients:        clients,
		currentIndex:   0,
		mu:             sync.RWMutex{},
		rpcTimeout:     rpcTimeout,
		logger:         logger,
		maxRetries:     types.MaxRetryCount,
		retryInterval:  1 * time.Second,
		lastScoreReset: now,
	}, nil
}

// GetCurrentClient returns the current RPC client info
func (p *RPCPool) GetCurrentClient() *RPCClientInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.clients[p.currentIndex]
}

// MoveToNextHealthyClient moves to the next healthy RPC client, returns nil if none available
func (p *RPCPool) MoveToNextHealthyClient() *RPCClientInfo {
	p.mu.Lock()
	defer p.mu.Unlock()

	startIndex := p.currentIndex
	for i := 0; i < len(p.clients); i++ {
		p.currentIndex = (p.currentIndex + 1) % len(p.clients)
		client := p.clients[p.currentIndex]
		if client.healthy && client.client != nil {
			p.logger.Info("Switching to next healthy RPC endpoint", zap.String("endpoint", client.endpoint))
			return client
		}
	}

	// Reset to original position if no healthy clients found
	p.currentIndex = startIndex
	return nil
}

// MarkClientUnhealthy marks the current client as unhealthy
func (p *RPCPool) MarkClientUnhealthy(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	client := p.clients[p.currentIndex]
	client.healthy = false
	client.lastError = err
	client.lastCheck = time.Now()

	p.logger.Warn("Marked RPC client as unhealthy",
		zap.String("endpoint", client.endpoint),
		zap.Error(err))
}

// GetHealthyClientCount returns the number of healthy clients
func (p *RPCPool) GetHealthyClientCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, client := range p.clients {
		if client.healthy && client.client != nil {
			count++
		}
	}
	return count
}

// updateScore updates the score for a client based on success or failure
func (p *RPCPool) updateScore(clientInfo *RPCClientInfo, success bool, isTimeout bool) {
	if success {
		clientInfo.successCount++
		// Update score on every success
		clientInfo.score += ScoreIncreaseOnSuccess
		if clientInfo.score > MaxScore {
			clientInfo.score = MaxScore
		}
	} else {
		clientInfo.failureCount++
		if isTimeout {
			clientInfo.timeoutCount++
			clientInfo.score -= ScoreDecayOnTimeout
		} else {
			clientInfo.score -= ScoreDecayOnFailure
		}
		if clientInfo.score < MinScore {
			clientInfo.score = MinScore
		}
	}
}

// UpdateScoreOnSuccess updates the score for the current client on successful request
func (p *RPCPool) UpdateScoreOnSuccess() {
	p.mu.Lock()
	defer p.mu.Unlock()

	currentClient := p.clients[p.currentIndex]
	p.updateScore(currentClient, true, false)

	p.logger.Debug("Updated endpoint score on success",
		zap.String("endpoint", currentClient.endpoint),
		zap.Float64("score", currentClient.score),
		zap.Int64("success_count", currentClient.successCount))
}

// UpdateScoreOnFailure updates the score for the current client on failed request
func (p *RPCPool) UpdateScoreOnFailure(err error, isTimeout bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	currentClient := p.clients[p.currentIndex]
	p.updateScore(currentClient, false, isTimeout)

	p.logger.Debug("Updated endpoint score on failure",
		zap.String("endpoint", currentClient.endpoint),
		zap.Float64("score", currentClient.score),
		zap.Int64("failure_count", currentClient.failureCount),
		zap.Int64("timeout_count", currentClient.timeoutCount),
		zap.Bool("is_timeout", isTimeout),
		zap.Error(err))
}

// TryRecoverUnhealthyClients attempts to recover all unhealthy clients
func (p *RPCPool) TryRecoverUnhealthyClients() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, client := range p.clients {
		if !client.healthy && client.client != nil {
			// Only attempt recovery if enough time has passed since last check
			if time.Since(client.lastCheck) > p.retryInterval {
				// Mark client as healthy again for retry - net/http handles connection recovery
				client.healthy = true
				client.lastError = nil
				client.lastCheck = time.Now()

				p.logger.Info("Marked RPC client as healthy for retry",
					zap.String("endpoint", client.endpoint))
			}
		}
	}
}

// ResetScoresIfNeeded resets all endpoint scores if enough time has passed
func (p *RPCPool) ResetScoresIfNeeded() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	if now.Sub(p.lastScoreReset) >= ScoreResetInterval {
		p.resetAllScores(now)
		p.lastScoreReset = now

		p.logger.Info("Reset all endpoint scores",
			zap.Duration("interval", ScoreResetInterval),
			zap.Int("endpoint_count", len(p.clients)))
	}
}

// resetAllScores resets scores for all endpoints (must be called with lock held)
func (p *RPCPool) resetAllScores(now time.Time) {
	for _, client := range p.clients {
		client.score = DefaultInitialScore
		client.successCount = 0
		client.failureCount = 0
		client.timeoutCount = 0
		client.lastReset = now

		p.logger.Debug("Reset endpoint score",
			zap.String("endpoint", client.endpoint),
			zap.Float64("score", client.score))
	}
}

// getSortedClientsByScore returns a copy of clients sorted by score (highest first)
func (p *RPCPool) getSortedClientsByScore() []*RPCClientInfo {
	// Create a copy of the clients slice
	sortedClients := make([]*RPCClientInfo, len(p.clients))
	copy(sortedClients, p.clients)

	// Sort by score (highest first), then by endpoint name for consistency
	sort.Slice(sortedClients, func(i, j int) bool {
		if sortedClients[i].score == sortedClients[j].score {
			return sortedClients[i].endpoint < sortedClients[j].endpoint
		}
		return sortedClients[i].score > sortedClients[j].score
	})

	return sortedClients
}

// logEndpointAttempt logs the attempt to use an RPC endpoint
func (p *RPCPool) logEndpointAttempt(endpoint string, retryAttempt int) {
	if retryAttempt == 0 {
		p.logger.Debug("Trying RPC endpoint", zap.String("endpoint", endpoint))
	} else {
		p.logger.Debug("Retrying RPC endpoint",
			zap.String("endpoint", endpoint),
			zap.Int("retry", retryAttempt))
	}
}

// logEndpointFailure logs the failure of an RPC endpoint
func (p *RPCPool) logEndpointFailure(endpoint string, err error, retryAttempt int) {
	if retryAttempt == 0 {
		p.logger.Warn("RPC request failed, trying next endpoint",
			zap.String("endpoint", endpoint),
			zap.String("error", err.Error()))
	} else {
		p.logger.Warn("RPC request failed during retry, trying next endpoint",
			zap.String("endpoint", endpoint),
			zap.String("error", err.Error()),
			zap.Int("retry", retryAttempt))
	}
}

// tryAllEndpointsWithScoring tries all endpoints prioritized by score until one succeeds or all fail
func (p *RPCPool) tryAllEndpointsWithScoring(ctx context.Context, fn func(context.Context) error, retryAttempt int) error {
	p.mu.Lock()
	sortedClients := p.getSortedClientsByScore()
	p.mu.Unlock()

	var lastErr error
	var isTimeout bool

	// Try endpoints in order of their scores (highest first)
	for _, client := range sortedClients {
		// Take a snapshot of client fields under read lock to avoid data races
		p.mu.RLock()
		isHealthy := client.healthy
		clientPtr := client.client
		p.mu.RUnlock()
		
		// Skip unhealthy clients
		if !isHealthy || clientPtr == nil {
			continue
		}

		// Update current index to this client
		p.mu.Lock()
		for i, c := range p.clients {
			if c == client {
				p.currentIndex = i
				break
			}
		}
		p.mu.Unlock()

		// Create a timeout context
		timeoutCtx, cancel := context.WithTimeout(ctx, p.rpcTimeout)

		p.logEndpointAttempt(client.endpoint, retryAttempt)

		err := fn(timeoutCtx)
		cancel()

		// Check if error is due to timeout
		isTimeout = err != nil && (errors.Is(timeoutCtx.Err(), context.DeadlineExceeded))

		if err == nil {
			p.UpdateScoreOnSuccess()
			return nil
		}

		// Failure - update score negatively and mark as unhealthy
		p.UpdateScoreOnFailure(err, isTimeout)
		p.MarkClientUnhealthy(err)
		lastErr = err
		p.logEndpointFailure(client.endpoint, err, retryAttempt)
	}

	// If no healthy clients were available, return appropriate error
	if lastErr == nil {
		return fmt.Errorf("no healthy RPC clients available")
	}

	return lastErr
}

// ExecuteWithFallback executes the given function with fallback to other endpoints if it fails
// and retries with exponential backoff if all endpoints fail
func (p *RPCPool) ExecuteWithFallback(ctx context.Context, fn func(context.Context) error) error {
	// Reset scores periodically to allow recovered endpoints to regain priority
	p.ResetScoresIfNeeded()

	// Try to recover unhealthy clients before starting
	p.TryRecoverUnhealthyClients()

	// First attempt: try all endpoints once using score-based prioritization
	err := p.tryAllEndpointsWithScoring(ctx, fn, 0)
	if err == nil {
		return nil
	}

	// If all endpoints failed, retry with exponential backoff using SleepWithRetry
	var lastErr error
	for retry := 1; retry <= p.maxRetries; retry++ {
		p.logger.Info("All RPC endpoints failed, retrying after backoff",
			zap.Int("retry", retry),
			zap.Int("max_retries", p.maxRetries),
			zap.Int("healthy_clients", p.GetHealthyClientCount()))

		// Use SleepWithRetry for exponential backoff with jitter
		if canceled := types.SleepWithRetry(ctx, retry); canceled {
			return ctx.Err()
		}

		// Try to recover unhealthy clients before retrying
		p.TryRecoverUnhealthyClients()

		// Try all endpoints again with score-based prioritization
		err := p.tryAllEndpointsWithScoring(ctx, fn, retry)
		if err == nil {
			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("all RPC endpoints failed after %d retries: %w", p.maxRetries, lastErr)
}

// CreateRPCClient creates a new RPC client with the given codec and RPC addresses
func CreateRPCClient(ctx types.Context, cdc codec.Codec, rpcAddresses []string, logger *zap.Logger) (*RPCClient, error) {
	if len(rpcAddresses) == 0 {
		return nil, errors.New("no RPC addresses provided")
	}

	// Create RPC pool with persistent HTTP clients
	pool, err := NewRPCPool(ctx, rpcAddresses, logger)
	if err != nil {
		return nil, err
	}

	// Get the first healthy client from the pool
	currentClient := pool.GetCurrentClient()
	if currentClient == nil || currentClient.client == nil {
		// Try to find any healthy client
		healthyClient := pool.MoveToNextHealthyClient()
		if healthyClient == nil {
			return nil, errors.New("no healthy RPC clients available")
		}
		currentClient = healthyClient
	}

	// Create RPC client with the healthy HTTP client from pool
	rpcClient := &RPCClient{
		HTTP: currentClient.client,
		cdc:  cdc,
		pool: pool,
	}

	return rpcClient, nil
}
