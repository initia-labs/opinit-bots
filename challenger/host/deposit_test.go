package host

import (
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

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestInitializeDepositHandler(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)
	bridgeInfo := ophosttypes.QueryBridgeResponse{
		BridgeId: 1,
	}
	hostNode := node.NewTestNode(nodetypes.NodeConfig{}, db.WithPrefix([]byte("test_host")), nil, nil, nil, nil)
	h := Host{
		BaseHost:   hostprovider.NewTestBaseHost(0, hostNode, bridgeInfo, nodetypes.NodeConfig{}, nil),
		eventQueue: make([]challengertypes.ChallengeEvent, 0),
	}

	cases := []struct {
		name              string
		initialL1Sequence uint64
		eventHandlerArgs  nodetypes.EventHandlerArgs
		expected          []challengertypes.ChallengeEvent
	}{
		{
			name:              "success",
			initialL1Sequence: 0,
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				BlockHeight:     1,
				BlockTime:       time.Unix(0, 100),
				LatestHeight:    1,
				TxIndex:         0,
				EventAttributes: hostprovider.InitiateTokenDepositEvents(1, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l1Denom", 100), []byte("databytes"), 1, "l2denom"),
			},
			expected: []challengertypes.ChallengeEvent{
				&challengertypes.Deposit{
					EventType:     "Deposit",
					Sequence:      1,
					L1BlockHeight: 1,
					From:          "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
					To:            "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
					L1Denom:       "l1Denom",
					Amount:        "100l2denom",
					Time:          time.Unix(0, 100),
					Timeout:       false,
				},
			},
		},
		{
			name:              "another bridge id",
			initialL1Sequence: 0,
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				BlockHeight:     1,
				BlockTime:       time.Unix(0, 100),
				LatestHeight:    1,
				TxIndex:         0,
				EventAttributes: hostprovider.InitiateTokenDepositEvents(2, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l1Denom", 100), []byte("databytes"), 1, "l2denom"),
			},
			expected: []challengertypes.ChallengeEvent{},
		},
		{
			name:              "processed l1 sequence",
			initialL1Sequence: 2,
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				BlockHeight:     1,
				BlockTime:       time.Unix(0, 100),
				LatestHeight:    1,
				TxIndex:         0,
				EventAttributes: hostprovider.InitiateTokenDepositEvents(1, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l1Denom", 100), []byte("databytes"), 1, "l2denom"),
			},
			expected: []challengertypes.ChallengeEvent{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h.initialL1Sequence = tc.initialL1Sequence

			err := h.initiateDepositHandler(types.Context{}, tc.eventHandlerArgs)
			require.NoError(t, err)
			require.Equal(t, tc.expected, h.eventQueue)

			h.eventQueue = make([]challengertypes.ChallengeEvent, 0)
		})
	}
	require.NoError(t, err)
}
