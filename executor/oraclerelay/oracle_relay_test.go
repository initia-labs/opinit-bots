package oraclerelay

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"cosmossdk.io/math"

	connecttypes "github.com/skip-mev/connect/v2/pkg/types"
	oracletypes "github.com/skip-mev/connect/v2/x/oracle/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"github.com/initia-labs/opinit-bots/types"
)

// mockHostNode implements the hostNode interface for testing
type mockHostNode struct {
	chainID            string
	bridgeInfo         ophosttypes.QueryBridgeResponse
	oracleEnabled      bool
	oraclePriceHash    *hostprovider.OraclePriceHashWithProof
	currencyPairs      []connecttypes.CurrencyPair
	oraclePrices       map[string]*oracletypes.GetPriceResponse
	queryPriceErr      map[string]error
	queryPriceHashErr  error
	queryCurrencyErr   error
}

func newMockHostNode(chainID string, oracleEnabled bool) *mockHostNode {
	return &mockHostNode{
		chainID:       chainID,
		oracleEnabled: oracleEnabled,
		bridgeInfo: ophosttypes.QueryBridgeResponse{
			BridgeId: 1,
			BridgeConfig: ophosttypes.BridgeConfig{
				OracleEnabled: oracleEnabled,
			},
		},
		oraclePrices:  make(map[string]*oracletypes.GetPriceResponse),
		queryPriceErr: make(map[string]error),
	}
}

func (m *mockHostNode) ChainId() string {
	return m.chainID
}

func (m *mockHostNode) BridgeInfo() ophosttypes.QueryBridgeResponse {
	return m.bridgeInfo
}

func (m *mockHostNode) OracleEnabled() bool {
	return m.oracleEnabled
}

func (m *mockHostNode) QueryOraclePriceHashWithProof(_ context.Context, _ uint64) (*hostprovider.OraclePriceHashWithProof, error) {
	if m.queryPriceHashErr != nil {
		return nil, m.queryPriceHashErr
	}
	return m.oraclePriceHash, nil
}

func (m *mockHostNode) QueryAllCurrencyPairs(_ context.Context) ([]connecttypes.CurrencyPair, error) {
	if m.queryCurrencyErr != nil {
		return nil, m.queryCurrencyErr
	}
	return m.currencyPairs, nil
}

func (m *mockHostNode) QueryOraclePrice(_ context.Context, base, quote string, _ int64) (*oracletypes.GetPriceResponse, error) {
	key := base + "/" + quote
	if err, ok := m.queryPriceErr[key]; ok && err != nil {
		return nil, err
	}
	return m.oraclePrices[key], nil
}

var _ hostNode = (*mockHostNode)(nil)

// mockChildNode implements the childNode interface for testing
type mockChildNode struct {
	l1ClientID          string
	latestRevisionHeight uint64
	broadcastedMsgs     []sdk.Msg
	queryL1ClientIDErr  error
	queryRevisionErr    error
	mu                  sync.Mutex
}

func newMockChildNode(l1ClientID string, latestRevisionHeight uint64) *mockChildNode {
	return &mockChildNode{
		l1ClientID:          l1ClientID,
		latestRevisionHeight: latestRevisionHeight,
		broadcastedMsgs:     make([]sdk.Msg, 0),
	}
}

func (m *mockChildNode) BroadcastMsgs(msgs []sdk.Msg, _ string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.broadcastedMsgs = append(m.broadcastedMsgs, msgs...)
}

func (m *mockChildNode) QueryL1ClientID(_ context.Context) (string, error) {
	if m.queryL1ClientIDErr != nil {
		return "", m.queryL1ClientIDErr
	}
	return m.l1ClientID, nil
}

func (m *mockChildNode) QueryLatestRevisionHeight(_ context.Context, _ string) (uint64, error) {
	if m.queryRevisionErr != nil {
		return 0, m.queryRevisionErr
	}
	return m.latestRevisionHeight, nil
}

func (m *mockChildNode) GetBroadcastedMsgs() []sdk.Msg {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.broadcastedMsgs
}

var _ childNode = (*mockChildNode)(nil)

func TestParseRevisionFromChainID(t *testing.T) {
	or := &OracleRelay{}

	cases := []struct {
		name     string
		chainID  string
		expected uint64
		hasError bool
	}{
		{
			name:     "standard chain ID with revision",
			chainID:  "interwoven-1",
			expected: 1,
			hasError: false,
		},
		{
			name:     "chain ID with higher revision",
			chainID:  "initia-testnet-123",
			expected: 123,
			hasError: false,
		},
		{
			name:     "chain ID with revision 0",
			chainID:  "chain-0",
			expected: 0,
			hasError: false,
		},
		{
			name:     "chain ID with large revision",
			chainID:  "mainnet-999999",
			expected: 999999,
			hasError: false,
		},
		{
			name:     "chain ID without revision number",
			chainID:  "invalid",
			expected: 0,
			hasError: true,
		},
		{
			name:     "chain ID with non-numeric suffix",
			chainID:  "chain-abc",
			expected: 0,
			hasError: true,
		},
		{
			name:     "empty chain ID",
			chainID:  "",
			expected: 0,
			hasError: true,
		},
		{
			name:     "chain ID with multiple dashes",
			chainID:  "my-test-chain-42",
			expected: 42,
			hasError: false,
		},
		{
			name:     "chain ID ending with dash",
			chainID:  "chain-",
			expected: 0,
			hasError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := or.parseRevisionFromChainID(tc.chainID)
			if tc.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestQueryAllOraclePrices(t *testing.T) {
	ctx := types.NewContext(context.Background(), zap.NewNop(), "")

	createPrice := func(priceVal int64, decimals uint64, nonce uint64, id uint64, timestamp time.Time) *oracletypes.GetPriceResponse {
		return &oracletypes.GetPriceResponse{
			Price: &oracletypes.QuotePrice{
				Price:          math.NewInt(priceVal),
				BlockTimestamp: timestamp,
				BlockHeight:    100,
			},
			Decimals: decimals,
			Nonce:    nonce,
			Id:       id,
		}
	}

	cases := []struct {
		name           string
		currencyPairs  []connecttypes.CurrencyPair
		prices         map[string]*oracletypes.GetPriceResponse
		priceErrors    map[string]error
		expectedCount  int
		expectError    bool
	}{
		{
			name: "successful query all prices",
			currencyPairs: []connecttypes.CurrencyPair{
				{Base: "BTC", Quote: "USD"},
				{Base: "ETH", Quote: "USD"},
			},
			prices: map[string]*oracletypes.GetPriceResponse{
				"BTC/USD": createPrice(5000000000000, 8, 1, 1, time.Now()),
				"ETH/USD": createPrice(300000000000, 8, 1, 2, time.Now()),
			},
			priceErrors:   nil,
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "skip failed price query",
			currencyPairs: []connecttypes.CurrencyPair{
				{Base: "BTC", Quote: "USD"},
				{Base: "INVALID", Quote: "USD"},
				{Base: "ETH", Quote: "USD"},
			},
			prices: map[string]*oracletypes.GetPriceResponse{
				"BTC/USD": createPrice(5000000000000, 8, 1, 1, time.Now()),
				"ETH/USD": createPrice(300000000000, 8, 1, 2, time.Now()),
			},
			priceErrors: map[string]error{
				"INVALID/USD": context.DeadlineExceeded,
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "skip nil price data",
			currencyPairs: []connecttypes.CurrencyPair{
				{Base: "BTC", Quote: "USD"},
				{Base: "NILPRICE", Quote: "USD"},
			},
			prices: map[string]*oracletypes.GetPriceResponse{
				"BTC/USD":      createPrice(5000000000000, 8, 1, 1, time.Now()),
				"NILPRICE/USD": {Price: nil, Decimals: 8, Nonce: 1, Id: 3}, // nil price data
			},
			priceErrors:   nil,
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:           "empty currency pairs",
			currencyPairs:  []connecttypes.CurrencyPair{},
			prices:         nil,
			priceErrors:    nil,
			expectedCount:  0,
			expectError:    false,
		},
		{
			name: "all queries fail",
			currencyPairs: []connecttypes.CurrencyPair{
				{Base: "FAIL1", Quote: "USD"},
				{Base: "FAIL2", Quote: "USD"},
			},
			prices: nil,
			priceErrors: map[string]error{
				"FAIL1/USD": context.DeadlineExceeded,
				"FAIL2/USD": context.DeadlineExceeded,
			},
			expectedCount: 0,
			expectError:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockHost := newMockHostNode("test-chain-1", true)
			mockHost.oraclePrices = tc.prices
			if tc.priceErrors != nil {
				mockHost.queryPriceErr = tc.priceErrors
			}

			or := &OracleRelay{
				host: mockHost,
				cfg: executortypes.OracleRelayConfig{
					Enable:   true,
					Interval: 30,
				},
			}

			result, err := or.queryAllOraclePrices(ctx, tc.currencyPairs, 100)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Len(t, result, tc.expectedCount)
			}
		})
	}
}

func TestStatusFieldsConcurrency(t *testing.T) {
	or := NewOracleRelayV1(executortypes.OracleRelayConfig{
		Enable:   true,
		Interval: 30,
	})

	var wg sync.WaitGroup
	numGoroutines := 100
	iterations := 1000

	// concurrent writes to lastRelayedL1Height
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				or.SetLastRelayedL1Height(uint64(id*iterations + j))
				_ = or.GetLastRelayedL1Height()
			}
		}(i)
	}
	wg.Wait()

	// concurrent writes to lastRelayedTime
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				or.SetLastRelayedTime(time.Now())
				_ = or.GetLastRelayedTime()
			}
		}(i)
	}
	wg.Wait()

	// concurrent mixed reads and writes
	wg.Add(numGoroutines * 2)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				or.SetLastRelayedL1Height(uint64(id*iterations + j))
				or.SetLastRelayedTime(time.Now())
			}
		}(i)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = or.GetLastRelayedL1Height()
				_ = or.GetLastRelayedTime()
			}
		}(i)
	}
	wg.Wait()

	// if there are no race conditions, the test passes
	require.True(t, true)
}

func TestStartContextCancellation(t *testing.T) {
	or := NewOracleRelayV1(executortypes.OracleRelayConfig{
		Enable:   true,
		Interval: 1, // a second interval for a faster test
	})

	mockHost := newMockHostNode("test-chain-1", false) // oracle disabled to skip relay logic
	mockChild := newMockChildNode("07-tendermint-0", 100)

	err := or.Initialize(mockHost, mockChild, "init1test")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	botCtx := types.NewContext(ctx, zap.NewNop(), "")

	done := make(chan error, 1)
	go func() {
		done <- or.Start(botCtx)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	// wait for Start to return
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}

func TestStartDisabled(t *testing.T) {
	or := NewOracleRelayV1(executortypes.OracleRelayConfig{
		Enable:   false,
		Interval: 30,
	})

	ctx := types.NewContext(context.Background(), zap.NewNop(), "")

	// return immediately without error
	err := or.Start(ctx)
	require.NoError(t, err)
}

func TestInitialize(t *testing.T) {
	or := NewOracleRelayV1(executortypes.OracleRelayConfig{
		Enable:   true,
		Interval: 30,
	})

	mockHost := newMockHostNode("test-chain-1", true)
	mockChild := newMockChildNode("07-tendermint-0", 100)

	err := or.Initialize(mockHost, mockChild, "init1sender")
	require.NoError(t, err)
	require.NotNil(t, or.host)
	require.NotNil(t, or.child)
	require.Equal(t, "init1sender", or.sender)
}

func TestNewOracleRelayV1(t *testing.T) {
	cfg := executortypes.OracleRelayConfig{
		Enable:        true,
		Interval:      60,
		CurrencyPairs: []string{"BTC/USD", "ETH/USD"},
	}

	or := NewOracleRelayV1(cfg)
	require.NotNil(t, or)
	require.Equal(t, uint8(1), or.version)
	require.Equal(t, cfg, or.cfg)
}

func TestStatusGettersSetters(t *testing.T) {
	or := NewOracleRelayV1(executortypes.OracleRelayConfig{})

	or.SetLastRelayedL1Height(12345)
	require.Equal(t, uint64(12345), or.GetLastRelayedL1Height())

	or.SetLastRelayedL1Height(0)
	require.Equal(t, uint64(0), or.GetLastRelayedL1Height())

	or.SetLastRelayedL1Height(^uint64(0)) // max uint64
	require.Equal(t, ^uint64(0), or.GetLastRelayedL1Height())

	now := time.Now()
	or.SetLastRelayedTime(now)
	require.Equal(t, now, or.GetLastRelayedTime())

	zeroTime := time.Time{}
	or.SetLastRelayedTime(zeroTime)
	require.Equal(t, zeroTime, or.GetLastRelayedTime())
}
