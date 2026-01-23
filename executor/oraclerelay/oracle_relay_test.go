package oraclerelay

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"cosmossdk.io/math"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	cmtypes "github.com/cometbft/cometbft/types"
	connecttypes "github.com/skip-mev/connect/v2/pkg/types"
	oracletypes "github.com/skip-mev/connect/v2/x/oracle/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/keys"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"github.com/initia-labs/opinit-bots/types"
)

// mockHostNode implements the hostNode interface for testing
type mockHostNode struct {
	chainID           string
	bridgeInfo        ophosttypes.QueryBridgeResponse
	oracleEnabled     bool
	oraclePriceHash   *hostprovider.OraclePriceHashWithProof
	currencyPairs     []connecttypes.CurrencyPair
	oraclePrices      map[string]oracletypes.GetPriceResponse
	queryPricesErr    error
	queryPriceHashErr error
	queryCurrencyErr  error

	latestHeight         int64
	commit               *coretypes.ResultCommit
	validators           []*cmtypes.Validator
	block                *coretypes.ResultBlock
	queryCommitErr       error
	queryValidatorsErr   error
	queryLatestHeightErr error
	queryBlockErr        error
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
		oraclePrices: make(map[string]oracletypes.GetPriceResponse),
		latestHeight: 100,
		commit:       createMockCommit(100),
		validators:   createMockValidators(),
		block:        createMockBlock(100),
	}
}

func createMockCommit(height int64) *coretypes.ResultCommit {
	return &coretypes.ResultCommit{
		SignedHeader: cmtypes.SignedHeader{
			Header: &cmtypes.Header{
				Height:          height,
				ProposerAddress: []byte("proposer_address_bytes"),
			},
			Commit: &cmtypes.Commit{
				Height: height,
			},
		},
	}
}

func createMockValidators() []*cmtypes.Validator {
	pubKey := cmtypes.NewMockPV().PrivKey.PubKey()
	return []*cmtypes.Validator{
		{
			Address:          []byte("proposer_address_bytes"),
			PubKey:           pubKey,
			VotingPower:      100,
			ProposerPriority: 0,
		},
	}
}

func createMockBlock(height int64) *coretypes.ResultBlock {
	return &coretypes.ResultBlock{
		Block: &cmtypes.Block{
			Header: cmtypes.Header{
				Height:          height,
				ProposerAddress: []byte("proposer_address_bytes"),
			},
		},
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

func (m *mockHostNode) QueryOraclePrices(_ context.Context, currencyPairIds []string, _ int64) ([]oracletypes.GetPriceResponse, error) {
	if m.queryPricesErr != nil {
		return nil, m.queryPricesErr
	}
	result := make([]oracletypes.GetPriceResponse, 0, len(currencyPairIds))
	for _, cpId := range currencyPairIds {
		if price, ok := m.oraclePrices[cpId]; ok {
			result = append(result, price)
		}
	}
	return result, nil
}

func (m *mockHostNode) QueryCommit(_ context.Context, _ int64) (*coretypes.ResultCommit, error) {
	if m.queryCommitErr != nil {
		return nil, m.queryCommitErr
	}
	return m.commit, nil
}

func (m *mockHostNode) QueryValidators(_ context.Context, _ int64) ([]*cmtypes.Validator, error) {
	if m.queryValidatorsErr != nil {
		return nil, m.queryValidatorsErr
	}
	return m.validators, nil
}

func (m *mockHostNode) QueryLatestHeight(_ context.Context) (int64, error) {
	if m.queryLatestHeightErr != nil {
		return 0, m.queryLatestHeightErr
	}
	return m.latestHeight, nil
}

func (m *mockHostNode) QueryBlock(_ context.Context, _ int64) (*coretypes.ResultBlock, error) {
	if m.queryBlockErr != nil {
		return nil, m.queryBlockErr
	}
	return m.block, nil
}

var _ hostNode = (*mockHostNode)(nil)

// mockChildNode implements the childNode interface for testing
type mockChildNode struct {
	l1ClientID           string
	latestRevisionHeight uint64
	broadcastedMsgs      []sdk.Msg
	queryL1ClientIDErr   error
	queryRevisionErr     error
	baseAccountAddr      string
	baseAccountAddrErr   error
	mu                   sync.Mutex
}

func newMockChildNode(l1ClientID string, latestRevisionHeight uint64) *mockChildNode {
	return &mockChildNode{
		l1ClientID:           l1ClientID,
		latestRevisionHeight: latestRevisionHeight,
		broadcastedMsgs:      make([]sdk.Msg, 0),
		baseAccountAddr:      "init1cy0v3p2exf5qdylc3h6wjyq8hl8yf9q0z7f8zd",
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

func (m *mockChildNode) BaseAccountAddressString() (string, error) {
	if m.baseAccountAddrErr != nil {
		return "", m.baseAccountAddrErr
	}
	return m.baseAccountAddr, nil
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

	createPrice := func(priceVal int64, decimals uint64, nonce uint64, id uint64, timestamp time.Time) oracletypes.GetPriceResponse {
		return oracletypes.GetPriceResponse{
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
		name          string
		currencyPairs []connecttypes.CurrencyPair
		prices        map[string]oracletypes.GetPriceResponse
		queryErr      error
		expectedCount int
		expectError   bool
	}{
		{
			name: "successful query all prices",
			currencyPairs: []connecttypes.CurrencyPair{
				{Base: "BTC", Quote: "USD"},
				{Base: "ETH", Quote: "USD"},
			},
			prices: map[string]oracletypes.GetPriceResponse{
				"BTC/USD": createPrice(5000000000000, 8, 1, 1, time.Now()),
				"ETH/USD": createPrice(300000000000, 8, 1, 2, time.Now()),
			},
			queryErr:      nil,
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "price count mismatch returns error",
			currencyPairs: []connecttypes.CurrencyPair{
				{Base: "BTC", Quote: "USD"},
				{Base: "MISSING", Quote: "USD"},
				{Base: "ETH", Quote: "USD"},
			},
			prices: map[string]oracletypes.GetPriceResponse{
				"BTC/USD": createPrice(5000000000000, 8, 1, 1, time.Now()),
				"ETH/USD": createPrice(300000000000, 8, 1, 2, time.Now()),
			},
			queryErr:      nil,
			expectedCount: 0,
			expectError:   true,
		},
		{
			name:          "empty currency pairs",
			currencyPairs: []connecttypes.CurrencyPair{},
			prices:        nil,
			queryErr:      nil,
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "query error returns error",
			currencyPairs: []connecttypes.CurrencyPair{
				{Base: "BTC", Quote: "USD"},
			},
			prices:        nil,
			queryErr:      context.DeadlineExceeded,
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockHost := newMockHostNode("test-chain-1", true)
			mockHost.oraclePrices = tc.prices
			mockHost.queryPricesErr = tc.queryErr

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

	err := or.Initialize(mockHost, mockChild, "init1wlvk4e083pd2lnfwu4e499vp5px7s4hgxmujql")
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

	err := or.Initialize(mockHost, mockChild, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5")
	require.NoError(t, err)
	require.NotNil(t, or.host)
	require.NotNil(t, or.child)
	require.Equal(t, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", or.sender)
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

func TestRelayOnce(t *testing.T) {
	createPrice := func(priceVal int64, decimals uint64, nonce uint64, id uint64, timestamp time.Time) oracletypes.GetPriceResponse {
		return oracletypes.GetPriceResponse{
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
		name              string
		oracleEnabled     bool
		chainID           string
		l1ClientID        string
		revisionHeight    uint64
		currencyPairs     []string
		hostCurrencyPairs []connecttypes.CurrencyPair
		prices            map[string]oracletypes.GetPriceResponse
		oraclePriceHash   *hostprovider.OraclePriceHashWithProof
		queryL1ClientErr  error
		queryRevisionErr  error
		queryPriceHashErr error
		queryCurrencyErr  error
		expectError       bool
		expectSkip        bool
		errorContains     string
	}{
		{
			name:           "oracle disabled - skip relay",
			oracleEnabled:  false,
			chainID:        "test-chain-1",
			l1ClientID:     "07-tendermint-0",
			revisionHeight: 100,
			expectError:    false,
			expectSkip:     true,
		},
		{
			name:             "failed to get L1 client ID",
			oracleEnabled:    true,
			chainID:          "test-chain-1",
			queryL1ClientErr: context.DeadlineExceeded,
			expectError:      true,
			errorContains:    "failed to get L1 client ID",
		},
		{
			name:             "failed to get latest revision height",
			oracleEnabled:    true,
			chainID:          "test-chain-1",
			l1ClientID:       "07-tendermint-0",
			queryRevisionErr: context.DeadlineExceeded,
			expectError:      true,
			errorContains:    "failed to get latest consensus height",
		},
		{
			name:           "no consensus states available",
			oracleEnabled:  true,
			chainID:        "test-chain-1",
			l1ClientID:     "07-tendermint-0",
			revisionHeight: 0,
			expectError:    true,
			errorContains:  "no consensus states available",
		},
		{
			name:           "invalid chain ID format",
			oracleEnabled:  true,
			chainID:        "invalid-chain",
			l1ClientID:     "07-tendermint-0",
			revisionHeight: 100,
			expectError:    true,
			errorContains:  "failed to parse oracle relay revision",
		},
		{
			name:              "failed to query oracle price hash",
			oracleEnabled:     true,
			chainID:           "test-chain-1",
			l1ClientID:        "07-tendermint-0",
			revisionHeight:    100,
			queryPriceHashErr: context.DeadlineExceeded,
			expectError:       true,
			errorContains:     "failed to query oracle price hash",
		},
		{
			name:             "failed to query currency pairs",
			oracleEnabled:    true,
			chainID:          "test-chain-1",
			l1ClientID:       "07-tendermint-0",
			revisionHeight:   100,
			currencyPairs:    []string{},
			queryCurrencyErr: context.DeadlineExceeded,
			oraclePriceHash: &hostprovider.OraclePriceHashWithProof{
				OraclePriceHash: ophosttypes.OraclePriceHash{
					Hash:          []byte("test_hash"),
					L1BlockHeight: 99,
					L1BlockTime:   1000000000,
				},
				Proof:       []byte("test_proof"),
				QueryHeight: 99,
			},
			expectError:   true,
			errorContains: "failed to query currency pairs",
		},
		{
			name:           "oracle price count mismatch",
			oracleEnabled:  true,
			chainID:        "test-chain-1",
			l1ClientID:     "07-tendermint-0",
			revisionHeight: 100,
			currencyPairs:  []string{"BTC/USD"},
			prices:         map[string]oracletypes.GetPriceResponse{},
			oraclePriceHash: &hostprovider.OraclePriceHashWithProof{
				OraclePriceHash: ophosttypes.OraclePriceHash{
					Hash:          []byte("test_hash"),
					L1BlockHeight: 99,
					L1BlockTime:   1000000000,
				},
				Proof:       []byte("test_proof"),
				QueryHeight: 99,
			},
			expectError:   true,
			errorContains: "oracle price count mismatch",
		},
		{
			name:           "successful relay with configured currency pairs",
			oracleEnabled:  true,
			chainID:        "test-chain-1",
			l1ClientID:     "07-tendermint-0",
			revisionHeight: 100,
			currencyPairs:  []string{"BTC/USD", "ETH/USD"},
			prices: map[string]oracletypes.GetPriceResponse{
				"BTC/USD": createPrice(5000000000000, 8, 1, 1, time.Now()),
				"ETH/USD": createPrice(300000000000, 8, 1, 2, time.Now()),
			},
			oraclePriceHash: &hostprovider.OraclePriceHashWithProof{
				OraclePriceHash: ophosttypes.OraclePriceHash{
					Hash:          []byte("test_hash"),
					L1BlockHeight: 99,
					L1BlockTime:   1000000000,
				},
				Proof:       []byte("test_proof"),
				QueryHeight: 99,
			},
			expectError: false,
		},
		{
			name:           "successful relay with L1 currency pairs",
			oracleEnabled:  true,
			chainID:        "test-chain-1",
			l1ClientID:     "07-tendermint-0",
			revisionHeight: 100,
			currencyPairs:  []string{},
			hostCurrencyPairs: []connecttypes.CurrencyPair{
				{Base: "BTC", Quote: "USD"},
			},
			prices: map[string]oracletypes.GetPriceResponse{
				"BTC/USD": createPrice(5000000000000, 8, 1, 1, time.Now()),
			},
			oraclePriceHash: &hostprovider.OraclePriceHashWithProof{
				OraclePriceHash: ophosttypes.OraclePriceHash{
					Hash:          []byte("test_hash"),
					L1BlockHeight: 99,
					L1BlockTime:   1000000000,
				},
				Proof:       []byte("test_proof"),
				QueryHeight: 99,
			},
			expectError: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			unlock := keys.SetSDKConfigContext("init")
			defer unlock()

			mockHost := newMockHostNode(tc.chainID, tc.oracleEnabled)
			mockHost.oraclePriceHash = tc.oraclePriceHash
			mockHost.currencyPairs = tc.hostCurrencyPairs
			mockHost.oraclePrices = tc.prices
			mockHost.queryPriceHashErr = tc.queryPriceHashErr
			mockHost.queryCurrencyErr = tc.queryCurrencyErr

			mockChild := newMockChildNode(tc.l1ClientID, tc.revisionHeight)
			mockChild.queryL1ClientIDErr = tc.queryL1ClientErr
			mockChild.queryRevisionErr = tc.queryRevisionErr

			or := NewOracleRelayV1(executortypes.OracleRelayConfig{
				Enable:        true,
				Interval:      30,
				CurrencyPairs: tc.currencyPairs,
			})
			err := or.Initialize(mockHost, mockChild, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5")
			require.NoError(t, err)

			ctx := types.NewContext(context.Background(), zap.NewNop(), "")
			err = or.relayOnce(ctx)

			if tc.expectSkip {
				require.NoError(t, err)
				require.Empty(t, mockChild.GetBroadcastedMsgs())
				return
			}

			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					require.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, mockChild.GetBroadcastedMsgs())
				require.Greater(t, or.GetLastRelayedL1Height(), uint64(0))
			}
		})
	}
}

func TestOracleRelayConfigValidate(t *testing.T) {
	cases := []struct {
		name        string
		config      executortypes.OracleRelayConfig
		expectError bool
	}{
		{
			name: "valid config with default interval",
			config: executortypes.OracleRelayConfig{
				Enable:   true,
				Interval: 30,
			},
			expectError: false,
		},
		{
			name: "valid config with currency pairs",
			config: executortypes.OracleRelayConfig{
				Enable:        true,
				Interval:      60,
				CurrencyPairs: []string{"BTC/USD", "ETH/USD"},
			},
			expectError: false,
		},
		{
			name: "valid disabled config",
			config: executortypes.OracleRelayConfig{
				Enable:   false,
				Interval: 30,
			},
			expectError: false,
		},
		{
			name: "invalid zero interval",
			config: executortypes.OracleRelayConfig{
				Enable:   true,
				Interval: 0,
			},
			expectError: true,
		},
		{
			name: "invalid negative interval",
			config: executortypes.OracleRelayConfig{
				Enable:   true,
				Interval: -1,
			},
			expectError: true,
		},
		{
			name:        "default config is valid",
			config:      executortypes.DefaultOracleRelayConfig(),
			expectError: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCurrencyPairParsing(t *testing.T) {
	ctx := types.NewContext(context.Background(), zap.NewNop(), "")

	createPrice := func(priceVal int64) oracletypes.GetPriceResponse {
		return oracletypes.GetPriceResponse{
			Price: &oracletypes.QuotePrice{
				Price:          math.NewInt(priceVal),
				BlockTimestamp: time.Now(),
				BlockHeight:    100,
			},
			Decimals: 8,
			Nonce:    1,
			Id:       1,
		}
	}

	cases := []struct {
		name          string
		currencyPairs []string
		prices        map[string]oracletypes.GetPriceResponse
		expectError   bool
	}{
		{
			name:          "valid currency pairs",
			currencyPairs: []string{"BTC/USD", "ETH/USD", "ATOM/USD"},
			prices: map[string]oracletypes.GetPriceResponse{
				"BTC/USD":  createPrice(5000000000000),
				"ETH/USD":  createPrice(300000000000),
				"ATOM/USD": createPrice(1000000000),
			},
			expectError: false,
		},
		{
			name:          "invalid currency pair format ignored",
			currencyPairs: []string{"BTC/USD", "INVALID", "ETH-USD", "ATOM/USD"},
			prices: map[string]oracletypes.GetPriceResponse{
				"BTC/USD":  createPrice(5000000000000),
				"ATOM/USD": createPrice(1000000000),
			},
			expectError: false, // filtered to 2 valid pairs, 2 prices available - success
		},
		{
			name:          "all invalid formats",
			currencyPairs: []string{"INVALID", "ALSO-INVALID", "NOSLASH"},
			prices:        map[string]oracletypes.GetPriceResponse{},
			expectError:   true, // 0 valid pairs results in "no oracle prices retrieved" error
		},
		{
			name:          "empty string in pairs",
			currencyPairs: []string{"BTC/USD", "", "ETH/USD"},
			prices: map[string]oracletypes.GetPriceResponse{
				"BTC/USD": createPrice(5000000000000),
				"ETH/USD": createPrice(300000000000),
			},
			expectError: false, // filtered to 2 valid pairs, 2 prices available - success
		},
		{
			name:          "currency pair with extra slashes",
			currencyPairs: []string{"BTC/USD/EXTRA", "ETH/USD"},
			prices: map[string]oracletypes.GetPriceResponse{
				"ETH/USD": createPrice(300000000000),
			},
			expectError: false, // filtered to 1 valid pair, 1 price available - success
		},
		{
			name:          "single valid currency pair",
			currencyPairs: []string{"ETH/USD"},
			prices: map[string]oracletypes.GetPriceResponse{
				"ETH/USD": createPrice(300000000000),
			},
			expectError: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			unlock := keys.SetSDKConfigContext("init")
			defer unlock()

			mockHost := newMockHostNode("test-chain-1", true)
			mockHost.oraclePrices = tc.prices
			mockHost.oraclePriceHash = &hostprovider.OraclePriceHashWithProof{
				OraclePriceHash: ophosttypes.OraclePriceHash{
					Hash:          []byte("test_hash"),
					L1BlockHeight: 99,
					L1BlockTime:   1000000000,
				},
				Proof:       []byte("test_proof"),
				QueryHeight: 99,
			}

			mockChild := newMockChildNode("07-tendermint-0", 100)

			or := NewOracleRelayV1(executortypes.OracleRelayConfig{
				Enable:        true,
				Interval:      30,
				CurrencyPairs: tc.currencyPairs,
			})
			err := or.Initialize(mockHost, mockChild, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5")
			require.NoError(t, err)

			// attempting relayOnce and checking how many messages were broadcast
			err = or.relayOnce(ctx)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				msgs := mockChild.GetBroadcastedMsgs()
				// Always 2 messages: MsgUpdateClient + MsgExec
				require.Len(t, msgs, 2)
			}
		})
	}
}

func TestRelayOnceProofHeightEdgeCases(t *testing.T) {
	createPrice := func(priceVal int64) oracletypes.GetPriceResponse {
		return oracletypes.GetPriceResponse{
			Price: &oracletypes.QuotePrice{
				Price:          math.NewInt(priceVal),
				BlockTimestamp: time.Now(),
				BlockHeight:    100,
			},
			Decimals: 8,
			Nonce:    1,
			Id:       1,
		}
	}

	cases := []struct {
		name           string
		revisionHeight uint64
		expectError    bool
		errorContains  string
	}{
		{
			name:           "minimum valid height (1)",
			revisionHeight: 1,
			expectError:    false,
		},
		{
			name:           "normal height",
			revisionHeight: 1000,
			expectError:    false,
		},
		{
			name:           "large height",
			revisionHeight: 999999999,
			expectError:    false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			unlock := keys.SetSDKConfigContext("init")
			defer unlock()

			mockHost := newMockHostNode("test-chain-1", true)
			mockHost.latestHeight = int64(tc.revisionHeight)
			mockHost.commit = createMockCommit(int64(tc.revisionHeight))
			mockHost.oraclePrices = map[string]oracletypes.GetPriceResponse{
				"BTC/USD": createPrice(5000000000000),
			}
			mockHost.oraclePriceHash = &hostprovider.OraclePriceHashWithProof{
				OraclePriceHash: ophosttypes.OraclePriceHash{
					Hash:          []byte("test_hash"),
					L1BlockHeight: tc.revisionHeight - 1,
					L1BlockTime:   1000000000,
				},
				Proof:       []byte("test_proof"),
				QueryHeight: tc.revisionHeight - 1,
			}

			mockChild := newMockChildNode("07-tendermint-0", tc.revisionHeight)

			or := NewOracleRelayV1(executortypes.OracleRelayConfig{
				Enable:        true,
				Interval:      30,
				CurrencyPairs: []string{"BTC/USD"},
			})
			err := or.Initialize(mockHost, mockChild, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5")
			require.NoError(t, err)

			ctx := types.NewContext(context.Background(), zap.NewNop(), "")
			err = or.relayOnce(ctx)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					require.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDuplicateRelayPrevention(t *testing.T) {
	createPrice := func(priceVal int64) oracletypes.GetPriceResponse {
		return oracletypes.GetPriceResponse{
			Price: &oracletypes.QuotePrice{
				Price:          math.NewInt(priceVal),
				BlockTimestamp: time.Now(),
				BlockHeight:    100,
			},
			Decimals: 8,
			Nonce:    1,
			Id:       1,
		}
	}

	t.Run("skip relay when L1 height unchanged", func(t *testing.T) {
		unlock := keys.SetSDKConfigContext("init")
		defer unlock()

		mockHost := newMockHostNode("test-chain-1", true)
		mockHost.latestHeight = 100
		mockHost.commit = createMockCommit(100)
		mockHost.oraclePrices = map[string]oracletypes.GetPriceResponse{
			"BTC/USD": createPrice(5000000000000),
		}
		mockHost.oraclePriceHash = &hostprovider.OraclePriceHashWithProof{
			OraclePriceHash: ophosttypes.OraclePriceHash{
				Hash:          []byte("test_hash"),
				L1BlockHeight: 99,
				L1BlockTime:   1000000000,
			},
			Proof:       []byte("test_proof"),
			QueryHeight: 99,
		}

		mockChild := newMockChildNode("07-tendermint-0", 50) // trusted height < target height

		or := NewOracleRelayV1(executortypes.OracleRelayConfig{
			Enable:        true,
			Interval:      30,
			CurrencyPairs: []string{"BTC/USD"},
		})
		err := or.Initialize(mockHost, mockChild, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5")
		require.NoError(t, err)

		ctx := types.NewContext(context.Background(), zap.NewNop(), "")

		// first relay should succeed (2 msgs: MsgUpdateClient + MsgExec)
		err = or.relayOnce(ctx)
		require.NoError(t, err)
		require.Len(t, mockChild.GetBroadcastedMsgs(), 2)
		require.Equal(t, uint64(99), or.GetLastRelayedL1Height())

		// second relay with same L1 height should skip
		err = or.relayOnce(ctx)
		require.NoError(t, err)
		require.Len(t, mockChild.GetBroadcastedMsgs(), 2) // still 2, not 4
	})

	t.Run("relay when L1 height increases", func(t *testing.T) {
		unlock := keys.SetSDKConfigContext("init")
		defer unlock()

		mockHost := newMockHostNode("test-chain-1", true)
		mockHost.latestHeight = 100
		mockHost.commit = createMockCommit(100)
		mockHost.oraclePrices = map[string]oracletypes.GetPriceResponse{
			"BTC/USD": createPrice(5000000000000),
		}
		mockHost.oraclePriceHash = &hostprovider.OraclePriceHashWithProof{
			OraclePriceHash: ophosttypes.OraclePriceHash{
				Hash:          []byte("test_hash"),
				L1BlockHeight: 99,
				L1BlockTime:   1000000000,
			},
			Proof:       []byte("test_proof"),
			QueryHeight: 99,
		}

		mockChild := newMockChildNode("07-tendermint-0", 50)

		or := NewOracleRelayV1(executortypes.OracleRelayConfig{
			Enable:        true,
			Interval:      30,
			CurrencyPairs: []string{"BTC/USD"},
		})
		err := or.Initialize(mockHost, mockChild, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5")
		require.NoError(t, err)

		ctx := types.NewContext(context.Background(), zap.NewNop(), "")

		// first relay (2 msgs)
		err = or.relayOnce(ctx)
		require.NoError(t, err)
		require.Len(t, mockChild.GetBroadcastedMsgs(), 2)
		require.Equal(t, uint64(99), or.GetLastRelayedL1Height())

		// simulate new oracle data (new L1 height)
		mockHost.latestHeight = 101
		mockHost.commit = createMockCommit(101)
		mockHost.oraclePriceHash.QueryHeight = 100
		mockHost.oraclePriceHash.OraclePriceHash.L1BlockHeight = 100

		// second relay with new L1 height should succeed (2 more msgs)
		err = or.relayOnce(ctx)
		require.NoError(t, err)
		require.Len(t, mockChild.GetBroadcastedMsgs(), 4)
		require.Equal(t, uint64(100), or.GetLastRelayedL1Height())
	})

	t.Run("first relay always proceeds (zero lastRelayedL1Height)", func(t *testing.T) {
		unlock := keys.SetSDKConfigContext("init")
		defer unlock()

		mockHost := newMockHostNode("test-chain-1", true)
		mockHost.latestHeight = 100
		mockHost.commit = createMockCommit(100)
		mockHost.oraclePrices = map[string]oracletypes.GetPriceResponse{
			"BTC/USD": createPrice(5000000000000),
		}
		mockHost.oraclePriceHash = &hostprovider.OraclePriceHashWithProof{
			OraclePriceHash: ophosttypes.OraclePriceHash{
				Hash:          []byte("test_hash"),
				L1BlockHeight: 99,
				L1BlockTime:   1000000000,
			},
			Proof:       []byte("test_proof"),
			QueryHeight: 99,
		}

		mockChild := newMockChildNode("07-tendermint-0", 50)

		or := NewOracleRelayV1(executortypes.OracleRelayConfig{
			Enable:        true,
			Interval:      30,
			CurrencyPairs: []string{"BTC/USD"},
		})
		err := or.Initialize(mockHost, mockChild, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5")
		require.NoError(t, err)

		// verify initial state
		require.Equal(t, uint64(0), or.GetLastRelayedL1Height())

		ctx := types.NewContext(context.Background(), zap.NewNop(), "")

		// the first relay should succeed (2 msgs: MsgUpdateClient + MsgExec)
		err = or.relayOnce(ctx)
		require.NoError(t, err)
		require.Len(t, mockChild.GetBroadcastedMsgs(), 2)
		require.Equal(t, uint64(99), or.GetLastRelayedL1Height())
	})
}
