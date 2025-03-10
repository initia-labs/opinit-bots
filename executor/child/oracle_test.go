package child

import (
	"context"
	"strconv"
	"testing"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/node"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestUpdateOracleHandler(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	childNode := node.NewTestNode(nodetypes.NodeConfig{}, db.WithPrefix([]byte("test_child")), nil, nil, nil, nil)
	bridgeInfo := opchildtypes.BridgeInfo{
		BridgeId: 1,
	}

	ch := Child{
		BaseChild: childprovider.NewTestBaseChild(0, childNode, nil, bridgeInfo, nil, nodetypes.NodeConfig{}),
	}

	cases := []struct {
		name             string
		eventHandlerArgs nodetypes.EventHandlerArgs
		expected         func() (msg string, fields []zapcore.Field)
		err              bool
	}{
		{
			name: "success",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: childprovider.UpdateOracleEvents(1, "sender"),
			},
			expected: func() (msg string, fields []zapcore.Field) {
				msg = "update oracle"
				fields = []zapcore.Field{
					zap.Int64("l1_blockHeight", 1),
					zap.String("from", "sender"),
				}
				return msg, fields
			},
			err: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logger, observedLogs := logCapturer()
			ctx := types.NewContext(context.Background(), logger, "")

			err := ch.updateOracleHandler(ctx, tc.eventHandlerArgs)
			if !tc.err {
				require.NoError(t, err)
				if tc.expected != nil {
					logs := observedLogs.TakeAll()
					require.Len(t, logs, 1)

					expectedMsg, expectedFields := tc.expected()
					require.Equal(t, expectedMsg, logs[0].Message)
					require.Equal(t, expectedFields, logs[0].Context)

					expectedL1Height, err := strconv.ParseInt(tc.eventHandlerArgs.EventAttributes[0].Value, 10, 64)
					require.NoError(t, err)

					require.Equal(t, expectedL1Height, ch.lastUpdatedOracleL1Height)
				}
			} else {
				require.Error(t, err)
			}
		})
	}
	require.NoError(t, err)
}
