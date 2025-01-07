package host

import (
	"context"
	"encoding/base64"
	"testing"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/node"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
		expected         func() (msg string, fields []zapcore.Field)
		err              bool
	}{
		{
			name: "success",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: hostprovider.ProposeOutputEvents("proposer", 1, 2, 3, []byte("output_root")),
			},
			expected: func() (msg string, fields []zapcore.Field) {
				msg = "propose output"
				fields = []zapcore.Field{
					zap.Uint64("bridge_id", 1),
					zap.String("proposer", "proposer"),
					zap.Uint64("output_index", 2),
					zap.Int64("l2_block_number", 3),
					zap.String("output_root", base64.StdEncoding.EncodeToString([]byte("output_root"))),
				}
				return msg, fields
			},
			err: false,
		},
		{
			name: "different bridge id",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: hostprovider.ProposeOutputEvents("proposer", 2, 2, 3, []byte("output_root")),
			},
			expected: nil,
			err:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logger, observedLogs := logCapturer()
			ctx := types.NewContext(context.Background(), logger, "")

			err := h.proposeOutputHandler(ctx, tc.eventHandlerArgs)
			if !tc.err {
				require.NoError(t, err)
				if tc.expected != nil {
					logs := observedLogs.TakeAll()
					require.Len(t, logs, 1)

					expectedMsg, expectedFields := tc.expected()
					require.Equal(t, expectedMsg, logs[0].Message)
					require.Equal(t, expectedFields, logs[0].Context)
				}
			} else {
				require.Error(t, err)
			}
		})
	}
	require.NoError(t, err)
}

func TestFinalizeWithdrawalHandler(t *testing.T) {
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
		expected         func() (msg string, fields []zapcore.Field)
		err              bool
	}{
		{
			name: "success",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: hostprovider.FinalizeWithdrawalEvents(1, 2, 3, "from", "to", "l1Denom", "l2Denom", sdk.NewInt64Coin("uinit", 10000)),
			},
			expected: func() (msg string, fields []zapcore.Field) {
				msg = "finalize withdrawal"
				fields = []zapcore.Field{
					zap.Uint64("bridge_id", 1),
					zap.Uint64("output_index", 2),
					zap.Uint64("l2_sequence", 3),
					zap.String("from", "from"),
					zap.String("to", "to"),
					zap.String("l1_denom", "l1Denom"),
					zap.String("l2_denom", "l2Denom"),
					zap.String("amount", "10000uinit"),
				}
				return msg, fields
			},
			err: false,
		},
		{
			name: "different bridge id",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: hostprovider.FinalizeWithdrawalEvents(2, 2, 3, "from", "to", "l1Denom", "l2Denom", sdk.NewInt64Coin("uinit", 10000)),
			},
			expected: nil,
			err:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logger, observedLogs := logCapturer()
			ctx := types.NewContext(context.Background(), logger, "")

			err := h.finalizeWithdrawalHandler(ctx, tc.eventHandlerArgs)
			if !tc.err {
				require.NoError(t, err)
				if tc.expected != nil {
					logs := observedLogs.TakeAll()
					require.Len(t, logs, 1)

					expectedMsg, expectedFields := tc.expected()
					require.Equal(t, expectedMsg, logs[0].Message)
					require.Equal(t, expectedFields, logs[0].Context)
				}
			} else {
				require.Error(t, err)
			}
		})
	}
	require.NoError(t, err)
}
