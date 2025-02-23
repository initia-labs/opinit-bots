package host

import (
	"context"
	"testing"
	"time"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/node"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/sha3"
)

func TestOracleTxHandler(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)
	hostNode := node.NewTestNode(nodetypes.NodeConfig{}, db.WithPrefix([]byte("test_host")), nil, nil, nil, nil)

	h := Host{}

	oracleTxDataChecksum := sha3.Sum256([]byte("oracle_tx"))

	cases := []struct {
		name          string
		oracleEnabled bool
		blockHeight   int64
		blockTime     time.Time
		extCommitBz   []byte
		expected      []challengertypes.ChallengeEvent
	}{
		{
			name:          "oracle enabled",
			oracleEnabled: true,
			blockHeight:   3,
			blockTime:     time.Unix(0, 100).UTC(),
			extCommitBz:   []byte("oracle_tx"),
			expected: []challengertypes.ChallengeEvent{
				&challengertypes.Oracle{
					EventType: "Oracle",
					L1Height:  3,
					Data:      oracleTxDataChecksum[:],
					Time:      time.Unix(0, 100).UTC(),
					Timeout:   false,
				},
			},
		},
		{
			name:          "oracle disabled",
			oracleEnabled: false,
			blockHeight:   3,
			blockTime:     time.Unix(0, 100).UTC(),
			extCommitBz:   []byte("valid_oracle_tx"),
			expected:      []challengertypes.ChallengeEvent{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h.BaseHost = hostprovider.NewTestBaseHost(0, hostNode, ophosttypes.QueryBridgeResponse{
				BridgeId: 1,
				BridgeConfig: ophosttypes.BridgeConfig{
					OracleEnabled: tc.oracleEnabled,
				},
			}, nodetypes.NodeConfig{}, nil)

			h.oracleTxHandler(tc.blockHeight, tc.blockTime, tc.extCommitBz)

			require.Equal(t, h.eventQueue, tc.expected)
			h.eventQueue = make([]challengertypes.ChallengeEvent, 0)
		})
	}
	require.NoError(t, err)
}

func TestUpdateOracleConfigHandler(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)
	hostNode := node.NewTestNode(nodetypes.NodeConfig{}, db.WithPrefix([]byte("test_host")), nil, nil, nil, nil)

	h := Host{
		BaseHost: hostprovider.NewTestBaseHost(0, hostNode, ophosttypes.QueryBridgeResponse{
			BridgeId: 1,
			BridgeConfig: ophosttypes.BridgeConfig{
				OracleEnabled: false,
			},
		}, nodetypes.NodeConfig{}, nil),
	}

	cases := []struct {
		name                  string
		bridgeId              uint64
		oracleEnabled         bool
		expectedOracleEnabled bool
		err                   bool
	}{
		{
			name:                  "oracle enabled",
			bridgeId:              1,
			oracleEnabled:         true,
			expectedOracleEnabled: true,
		},
		{
			name:                  "oracle disabled",
			bridgeId:              1,
			oracleEnabled:         false,
			expectedOracleEnabled: false,
		},
		{
			name:                  "another bridge id",
			bridgeId:              2,
			oracleEnabled:         true,
			expectedOracleEnabled: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := h.updateOracleConfigHandler(types.NewContext(context.Background(), zap.NewNop(), ""), nodetypes.EventHandlerArgs{
				EventAttributes: hostprovider.UpdateOracleConfigEvents(tc.bridgeId, tc.oracleEnabled),
			})
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedOracleEnabled, h.OracleEnabled())
			}
		})
	}
}
