package rpcclient

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/initia-labs/opinit-bots/types"
)

// createTestRPCPool creates a test RPC pool with mock endpoints
func createTestRPCPool(t *testing.T, endpoints []string) *RPCPool {
	ctx := types.NewContext(context.Background(), zap.NewNop(), "")
	logger := zap.NewNop()

	pool, err := NewRPCPool(ctx, endpoints, logger)
	require.NoError(t, err)

	// Mark all clients as healthy for testing (since endpoints are fake)
	pool.mu.Lock()
	for _, client := range pool.clients {
		client.healthy = true
		// The scoring functions don't actually use the HTTP client
	}
	pool.mu.Unlock()

	return pool
}

func TestRPCPool_InitialScores(t *testing.T) {
	endpoints := []string{"http://endpoint1:26657", "http://endpoint2:26657", "http://endpoint3:26657"}
	pool := createTestRPCPool(t, endpoints)

	// Check that all endpoints start with default initial score
	pool.mu.RLock()
	for _, client := range pool.clients {
		assert.Equal(t, DefaultInitialScore, client.score, "Initial score should be DefaultInitialScore")
		assert.Equal(t, int64(0), client.successCount, "Initial success count should be 0")
		assert.Equal(t, int64(0), client.failureCount, "Initial failure count should be 0")
		assert.Equal(t, int64(0), client.timeoutCount, "Initial timeout count should be 0")
	}
	pool.mu.RUnlock()
}

func TestRPCPool_UpdateScoreOnSuccess(t *testing.T) {
	endpoints := []string{"http://endpoint1:26657", "http://endpoint2:26657"}
	pool := createTestRPCPool(t, endpoints)

	pool.mu.RLock()
	initialScore := pool.clients[0].score
	currentClient := pool.clients[pool.currentIndex]
	pool.mu.RUnlock()

	// Update score on success
	pool.UpdateScoreOnSuccess(currentClient)

	pool.mu.RLock()
	updatedClient := pool.clients[pool.currentIndex]
	assert.Equal(t, initialScore+ScoreIncreaseOnSuccess, updatedClient.score, "Score should increase on success")
	assert.Equal(t, int64(1), updatedClient.successCount, "Success count should increment")
	assert.Equal(t, int64(0), updatedClient.failureCount, "Failure count should remain 0")
	pool.mu.RUnlock()
}

func TestRPCPool_UpdateScoreOnFailure(t *testing.T) {
	endpoints := []string{"http://endpoint1:26657", "http://endpoint2:26657"}
	pool := createTestRPCPool(t, endpoints)

	pool.mu.RLock()
	initialScore := pool.clients[0].score
	currentClient := pool.clients[pool.currentIndex]
	pool.mu.RUnlock()
	testErr := fmt.Errorf("test error")

	// Test regular failure
	pool.UpdateScoreOnFailure(currentClient, testErr, false)

	pool.mu.RLock()
	updatedClient := pool.clients[pool.currentIndex]
	assert.Equal(t, initialScore-ScoreDecayOnFailure, updatedClient.score, "Score should decrease on failure")
	assert.Equal(t, int64(0), updatedClient.successCount, "Success count should remain 0")
	assert.Equal(t, int64(1), updatedClient.failureCount, "Failure count should increment")
	assert.Equal(t, int64(0), updatedClient.timeoutCount, "Timeout count should remain 0")
	pool.mu.RUnlock()
}

func TestRPCPool_UpdateScoreOnTimeout(t *testing.T) {
	endpoints := []string{"http://endpoint1:26657", "http://endpoint2:26657"}
	pool := createTestRPCPool(t, endpoints)

	pool.mu.RLock()
	initialScore := pool.clients[0].score
	currentClient := pool.clients[pool.currentIndex]
	pool.mu.RUnlock()
	testErr := fmt.Errorf("timeout error")

	// Test timeout failure
	pool.UpdateScoreOnFailure(currentClient, testErr, true)

	pool.mu.RLock()
	updatedClient := pool.clients[pool.currentIndex]
	assert.Equal(t, initialScore-ScoreDecayOnTimeout, updatedClient.score, "Score should decrease more on timeout")
	assert.Equal(t, int64(0), updatedClient.successCount, "Success count should remain 0")
	assert.Equal(t, int64(1), updatedClient.failureCount, "Failure count should increment")
	assert.Equal(t, int64(1), updatedClient.timeoutCount, "Timeout count should increment")
	pool.mu.RUnlock()
}

func TestRPCPool_ScoreBounds(t *testing.T) {
	endpoints := []string{"http://endpoint1:26657"}
	pool := createTestRPCPool(t, endpoints)

	// Test maximum score bound
	pool.mu.RLock()
	targetClient := pool.clients[0]
	pool.mu.RUnlock()
	
	for i := 0; i < 50; i++ {
		pool.UpdateScoreOnSuccess(targetClient)
	}

	pool.mu.RLock()
	assert.Equal(t, MaxScore, pool.clients[0].score, "Score should not exceed MaxScore")
	pool.mu.RUnlock()

	// Reset score to test minimum bound
	pool.mu.Lock()
	pool.clients[0].score = DefaultInitialScore
	pool.mu.Unlock()

	// Test minimum score bound
	testErr := fmt.Errorf("test error")
	for i := 0; i < 50; i++ {
		pool.UpdateScoreOnFailure(targetClient, testErr, false)
	}

	pool.mu.RLock()
	assert.Equal(t, MinScore, pool.clients[0].score, "Score should not go below MinScore")
	pool.mu.RUnlock()
}

func TestRPCPool_GetSortedClientsByScore(t *testing.T) {
	endpoints := []string{"http://endpoint1:26657", "http://endpoint2:26657", "http://endpoint3:26657"}
	pool := createTestRPCPool(t, endpoints)

	// Set different scores for each endpoint
	pool.mu.Lock()
	pool.clients[0].score = 150.0 // highest
	pool.clients[1].score = 80.0  // lowest
	pool.clients[2].score = 120.0 // middle
	pool.mu.Unlock()

	sortedClients := pool.getSortedClientsByScore()

	// Verify sorting (highest first)
	assert.Equal(t, 150.0, sortedClients[0].score, "First client should have highest score")
	assert.Equal(t, 120.0, sortedClients[1].score, "Second client should have middle score")
	assert.Equal(t, 80.0, sortedClients[2].score, "Third client should have lowest score")

	// Verify endpoints are correctly ordered
	assert.Equal(t, "http://endpoint1:26657", sortedClients[0].endpoint)
	assert.Equal(t, "http://endpoint3:26657", sortedClients[1].endpoint)
	assert.Equal(t, "http://endpoint2:26657", sortedClients[2].endpoint)
}

func TestRPCPool_GetBestHealthyClient(t *testing.T) {
	endpoints := []string{"http://endpoint1:26657", "http://endpoint2:26657", "http://endpoint3:26657"}
	pool := createTestRPCPool(t, endpoints)

	// Set different scores
	pool.mu.Lock()
	pool.clients[0].score = 80.0
	pool.clients[1].score = 150.0 // highest, should be selected
	pool.clients[2].score = 120.0
	pool.mu.Unlock()

	// Use getSortedClientsByScore and find the best healthy client
	sortedClients := pool.getSortedClientsByScore()
	var bestClient *RPCClientInfo
	for _, client := range sortedClients {
		if client.healthy && client.client != nil {
			bestClient = client
			break
		}
	}

	require.NotNil(t, bestClient, "Should return a client")
	assert.Equal(t, 150.0, bestClient.score, "Should return client with highest score")
	assert.Equal(t, "http://endpoint2:26657", bestClient.endpoint)
}

func TestRPCPool_GetBestHealthyClient_OnlyHealthy(t *testing.T) {
	endpoints := []string{"http://endpoint1:26657", "http://endpoint2:26657", "http://endpoint3:26657"}
	pool := createTestRPCPool(t, endpoints)

	// Set different scores and mark some as unhealthy
	pool.mu.Lock()
	pool.clients[0].score = 80.0
	pool.clients[0].healthy = false // unhealthy, should be skipped
	pool.clients[1].score = 150.0   // highest but unhealthy
	pool.clients[1].healthy = false // unhealthy, should be skipped
	pool.clients[2].score = 120.0   // healthy, should be selected
	pool.clients[2].healthy = true
	pool.mu.Unlock()

	// Use getSortedClientsByScore and find the best healthy client
	sortedClients := pool.getSortedClientsByScore()
	var bestClient *RPCClientInfo
	for _, client := range sortedClients {
		if client.healthy && client.client != nil {
			bestClient = client
			break
		}
	}

	require.NotNil(t, bestClient, "Should return a healthy client")
	assert.Equal(t, 120.0, bestClient.score, "Should return healthy client with highest score")
	assert.Equal(t, "http://endpoint3:26657", bestClient.endpoint)
}

func TestRPCPool_GetBestHealthyClient_NoHealthyClients(t *testing.T) {
	endpoints := []string{"http://endpoint1:26657", "http://endpoint2:26657"}
	pool := createTestRPCPool(t, endpoints)

	// Mark all clients as unhealthy
	pool.mu.Lock()
	for _, client := range pool.clients {
		client.healthy = false
	}
	pool.mu.Unlock()

	// Use getSortedClientsByScore and find the best healthy client
	sortedClients := pool.getSortedClientsByScore()
	var bestClient *RPCClientInfo
	for _, client := range sortedClients {
		if client.healthy && client.client != nil {
			bestClient = client
			break
		}
	}

	assert.Nil(t, bestClient, "Should return nil when no healthy clients available")
}

func TestRPCPool_ResetScores(t *testing.T) {
	endpoints := []string{"http://endpoint1:26657", "http://endpoint2:26657"}
	pool := createTestRPCPool(t, endpoints)

	// Modify scores and counters
	pool.mu.Lock()
	pool.clients[0].score = 150.0
	pool.clients[0].successCount = 10
	pool.clients[0].failureCount = 5
	pool.clients[0].timeoutCount = 2
	pool.clients[1].score = 80.0
	pool.clients[1].successCount = 3
	pool.clients[1].failureCount = 8
	pool.clients[1].timeoutCount = 4
	pool.mu.Unlock()

	// Reset scores
	now := time.Now()
	pool.mu.Lock()
	pool.resetAllScores(now)
	pool.mu.Unlock()

	// Verify reset
	pool.mu.RLock()
	for _, client := range pool.clients {
		assert.Equal(t, DefaultInitialScore, client.score, "Score should be reset to default")
		assert.Equal(t, int64(0), client.successCount, "Success count should be reset to 0")
		assert.Equal(t, int64(0), client.failureCount, "Failure count should be reset to 0")
		assert.Equal(t, int64(0), client.timeoutCount, "Timeout count should be reset to 0")
		assert.Equal(t, now, client.lastReset, "Last reset time should be updated")
	}
	pool.mu.RUnlock()
}

func TestRPCPool_ResetScoresIfNeeded(t *testing.T) {
	endpoints := []string{"http://endpoint1:26657"}
	pool := createTestRPCPool(t, endpoints)

	// Set last reset time to past the interval
	pool.mu.Lock()
	pool.lastScoreReset = time.Now().Add(-ScoreResetInterval - time.Minute)
	pool.clients[0].score = 150.0
	pool.mu.Unlock()

	// Call ResetScoresIfNeeded
	pool.ResetScoresIfNeeded()

	// Verify scores were reset
	pool.mu.RLock()
	assert.Equal(t, DefaultInitialScore, pool.clients[0].score, "Score should be reset")
	assert.True(t, time.Since(pool.lastScoreReset) < time.Minute, "Last reset time should be updated")
	pool.mu.RUnlock()
}

func TestRPCPool_ResetScoresIfNeeded_NotYetTime(t *testing.T) {
	endpoints := []string{"http://endpoint1:26657"}
	pool := createTestRPCPool(t, endpoints)

	// Set last reset time to recent
	pool.mu.Lock()
	pool.lastScoreReset = time.Now().Add(-time.Minute) // Only 1 minute ago
	pool.clients[0].score = 150.0
	pool.mu.Unlock()

	// Call ResetScoresIfNeeded
	pool.ResetScoresIfNeeded()

	// Verify scores were NOT reset
	pool.mu.RLock()
	assert.Equal(t, 150.0, pool.clients[0].score, "Score should not be reset yet")
	pool.mu.RUnlock()
}

func TestRPCPool_TryAllEndpointsWithScoring(t *testing.T) {
	endpoints := []string{"http://endpoint1:26657", "http://endpoint2:26657", "http://endpoint3:26657"}
	pool := createTestRPCPool(t, endpoints)

	// Set different scores
	pool.mu.Lock()
	pool.clients[0].score = 80.0  // lowest
	pool.clients[1].score = 150.0 // highest
	pool.clients[2].score = 120.0 // middle
	pool.mu.Unlock()

	callOrder := []string{}

	// Test function that records which endpoint was called
	testFn := func(ctx context.Context) error {
		currentEndpoint := pool.GetCurrentClient().endpoint
		callOrder = append(callOrder, currentEndpoint)

		// Fail for all except the highest scored endpoint
		if currentEndpoint == "http://endpoint2:26657" {
			return nil // Success for highest scored endpoint
		}
		return fmt.Errorf("endpoint %s failed", currentEndpoint)
	}

	err := pool.tryAllEndpointsWithScoring(context.Background(), testFn, 0)

	assert.NoError(t, err, "Should succeed when highest scored endpoint works")

	// Verify that endpoints were tried in score order (highest first)
	require.Len(t, callOrder, 1, "Should only call one endpoint (the successful one)")
	assert.Equal(t, "http://endpoint2:26657", callOrder[0], "Should try highest scored endpoint first")
}

func TestRPCPool_TryAllEndpointsWithScoring_AllFail(t *testing.T) {
	endpoints := []string{"http://endpoint1:26657", "http://endpoint2:26657"}
	pool := createTestRPCPool(t, endpoints)

	// Set different scores
	pool.mu.Lock()
	pool.clients[0].score = 80.0
	pool.clients[1].score = 150.0
	pool.mu.Unlock()

	callOrder := []string{}

	// Test function that always fails but records call order
	testFn := func(ctx context.Context) error {
		currentEndpoint := pool.GetCurrentClient().endpoint
		callOrder = append(callOrder, currentEndpoint)
		return fmt.Errorf("endpoint %s failed", currentEndpoint)
	}

	err := pool.tryAllEndpointsWithScoring(context.Background(), testFn, 0)

	assert.Error(t, err, "Should fail when all endpoints fail")

	// Verify that endpoints were tried in score order (highest first)
	require.Len(t, callOrder, 2, "Should try both endpoints")
	assert.Equal(t, "http://endpoint2:26657", callOrder[0], "Should try highest scored endpoint first")
	assert.Equal(t, "http://endpoint1:26657", callOrder[1], "Should try lower scored endpoint second")
}

func TestRPCPool_ScoreInflationPrevention(t *testing.T) {
	endpoints := []string{"http://endpoint1:26657"}
	pool := createTestRPCPool(t, endpoints)

	pool.mu.RLock()
	initialScore := pool.clients[0].score
	targetClient := pool.clients[0]
	pool.mu.RUnlock()

	// Test multiple successful requests - score should only increase once
	pool.UpdateScoreOnSuccess(targetClient)
	pool.mu.RLock()
	firstSuccessScore := pool.clients[0].score
	firstSuccessCount := pool.clients[0].successCount
	pool.mu.RUnlock()

	assert.Equal(t, initialScore+ScoreIncreaseOnSuccess, firstSuccessScore, "Score should increase on first success")
	assert.Equal(t, int64(1), firstSuccessCount, "Success count should be 1")

	// Additional successful requests should continue to increase score
	for i := 0; i < 5; i++ {
		pool.UpdateScoreOnSuccess(targetClient)
	}

	pool.mu.RLock()
	finalScore := pool.clients[0].score
	finalSuccessCount := pool.clients[0].successCount
	pool.mu.RUnlock()

	expectedFinalScore := initialScore + (6 * ScoreIncreaseOnSuccess) // 6 total successes
	assert.Equal(t, expectedFinalScore, finalScore, "Score should increase with each success")
	assert.Equal(t, int64(6), finalSuccessCount, "Success count should continue incrementing")

	// Test that failure resets the success state
	testErr := fmt.Errorf("test error")
	pool.UpdateScoreOnFailure(targetClient, testErr, false)

	pool.mu.RLock()
	afterFailureScore := pool.clients[0].score
	pool.mu.RUnlock()

	assert.Equal(t, expectedFinalScore-ScoreDecayOnFailure, afterFailureScore, "Score should decrease on failure")

	// Test that score can increase again after failure
	pool.UpdateScoreOnSuccess(targetClient)

	pool.mu.RLock()
	recoveryScore := pool.clients[0].score
	pool.mu.RUnlock()

	assert.Equal(t, afterFailureScore+ScoreIncreaseOnSuccess, recoveryScore, "Score should increase on success after failure")
}

func TestRPCPool_ConcurrentScoreUpdates(t *testing.T) {
	endpoints := []string{"http://endpoint1:26657"}
	pool := createTestRPCPool(t, endpoints)

	pool.mu.RLock()
	targetClient := pool.clients[0]
	pool.mu.RUnlock()

	// Run concurrent score updates
	done := make(chan bool, 2)

	// Goroutine 1: Success updates
	go func() {
		for i := 0; i < 50; i++ {
			pool.UpdateScoreOnSuccess(targetClient)
		}
		done <- true
	}()

	// Goroutine 2: Failure updates
	go func() {
		testErr := fmt.Errorf("test error")
		for i := 0; i < 10; i++ {
			pool.UpdateScoreOnFailure(targetClient, testErr, false)
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify final state is consistent
	pool.mu.RLock()
	client := pool.clients[0]
	assert.Equal(t, int64(50), client.successCount, "Success count should be 50")
	assert.Equal(t, int64(10), client.failureCount, "Failure count should be 10")

	// Score should be initial + (50 * success) - (10 * failure), bounded by MinScore and MaxScore
	expectedScore := DefaultInitialScore + (50 * ScoreIncreaseOnSuccess) - (10 * ScoreDecayOnFailure)
	if expectedScore < MinScore {
		expectedScore = MinScore
	} else if expectedScore > MaxScore {
		expectedScore = MaxScore
	}
	assert.Equal(t, expectedScore, client.score, "Score should reflect all updates within bounds")

	// Also verify the score is within valid bounds
	assert.GreaterOrEqual(t, client.score, MinScore, "Score should not be below MinScore")
	assert.LessOrEqual(t, client.score, MaxScore, "Score should not exceed MaxScore")
	pool.mu.RUnlock()
}