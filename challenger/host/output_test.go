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
)

func TestProposeOutputHandler(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	hostNode := node.NewTestNode(nodetypes.NodeConfig{}, db.WithPrefix([]byte("test_host")), nil, nil, nil, nil)
	bridgeInfo := ophosttypes.QueryBridgeResponse{
		BridgeId: 1,
	}

	h := Host{
		BaseHost: hostprovider.NewTestBaseHost(0, hostNode, bridgeInfo, nodetypes.NodeConfig{}, nil),
	}

	cases := []struct {
		name             string
		eventHandlerArgs nodetypes.EventHandlerArgs
		expected         []challengertypes.ChallengeEvent
	}{
		{
			name: "success",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				BlockTime:       time.Unix(0, 100),
				EventAttributes: hostprovider.ProposeOutputEvents("proposer", 1, 2, 3, []byte("output_root")),
			},
			expected: []challengertypes.ChallengeEvent{
				&challengertypes.Output{
					EventType:     "Output",
					L2BlockNumber: 3,
					OutputIndex:   2,
					OutputRoot:    []byte("output_root"),
					Time:          time.Unix(0, 100),
					Timeout:       false,
				},
			},
		},
		{
			name: "different bridge id",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: hostprovider.ProposeOutputEvents("proposer", 2, 2, 3, []byte("output_root")),
			},
			expected: []challengertypes.ChallengeEvent{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := types.NewContext(context.Background(), zap.NewNop(), "")

			err := h.proposeOutputHandler(ctx, tc.eventHandlerArgs)
			require.NoError(t, err)

			require.Equal(t, tc.expected, h.outputPendingEventQueue)

			h.outputPendingEventQueue = make([]challengertypes.ChallengeEvent, 0)
		})
	}
	require.NoError(t, err)
}
