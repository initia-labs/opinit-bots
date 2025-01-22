package child

import (
	"context"
	"testing"
	"time"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/node"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/sha3"
)

func TestOracleTxHandler(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)
	childNode := node.NewTestNode(nodetypes.NodeConfig{}, db.WithPrefix([]byte("test_child")), nil, nil, nil, nil)
	bridgeInfo := ophosttypes.QueryBridgeResponse{
		BridgeId: 1,
	}

	ch := Child{
		BaseChild:  childprovider.NewTestBaseChild(0, childNode, nil, bridgeInfo, nil, nodetypes.NodeConfig{}),
		eventQueue: make([]challengertypes.ChallengeEvent, 0),
	}

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
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ch.oracleTxHandler(types.NewContext(context.Background(), zap.NewNop(), ""), tc.blockTime, "sender", tc.blockHeight, tc.extCommitBz)

			require.Equal(t, ch.eventQueue, tc.expected)
			ch.eventQueue = make([]challengertypes.ChallengeEvent, 0)
		})
	}
	require.NoError(t, err)
}
