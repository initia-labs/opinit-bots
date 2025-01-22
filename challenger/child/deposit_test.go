package child

import (
	"context"
	"strconv"
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

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestFinalizeDepositHandler(t *testing.T) {
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

	cases := []struct {
		name             string
		eventHandlerArgs nodetypes.EventHandlerArgs
		expected         []challengertypes.ChallengeEvent
		err              bool
	}{
		{
			name: "empty event queue",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				BlockTime:       time.Unix(0, 100).UTC(),
				EventAttributes: childprovider.FinalizeDepositEvents(1, "sender", "recipient", "denom", "baseDenom", sdk.NewInt64Coin("denom", 10000), 2),
			},
			expected: []challengertypes.ChallengeEvent{
				&challengertypes.Deposit{
					EventType:     "Deposit",
					Sequence:      1,
					L1BlockHeight: 2,
					From:          "sender",
					To:            "recipient",
					L1Denom:       "baseDenom",
					Amount:        "10000denom",
					Time:          time.Unix(0, 100).UTC(),
				},
			},
			err: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := types.NewContext(context.Background(), zap.NewNop(), "")

			err := ch.finalizeDepositHandler(ctx, tc.eventHandlerArgs)
			require.NoError(t, err)
			expectedL1Height, err := strconv.ParseInt(tc.eventHandlerArgs.EventAttributes[6].Value, 10, 64)
			require.NoError(t, err)
			expectedL1Sequence, err := strconv.ParseUint(tc.eventHandlerArgs.EventAttributes[0].Value, 10, 64)
			require.NoError(t, err)

			require.Equal(t, expectedL1Height, ch.lastFinalizedDepositL1BlockHeight)
			require.Equal(t, expectedL1Sequence, ch.lastFinalizedDepositL1Sequence)

			require.Equal(t, tc.expected, ch.eventQueue)
			ch.eventQueue = make([]challengertypes.ChallengeEvent, 0)
		})
	}
	require.NoError(t, err)
}
