package host

import (
	"context"
	"testing"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/node"
	"github.com/initia-labs/opinit-bots/node/broadcaster"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestRecordBatchHandler(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	cdc, txConfig, err := hostprovider.GetCodec("init")
	require.NoError(t, err)

	broadcaster, err := broadcaster.NewTestBroadcaster(cdc, db.WithPrefix([]byte("test_host")), nil, txConfig, "init", 1)
	require.NoError(t, err)

	batchSubmitter, err := broadcaster.AccountByIndex(0)
	require.NoError(t, err)

	hostNode := node.NewTestNode(nodetypes.NodeConfig{}, db.WithPrefix([]byte("test_host")), nil, nil, nil, broadcaster)
	bridgeInfo := ophosttypes.QueryBridgeResponse{
		BridgeId: 1,
	}

	h := Host{
		BaseHost: hostprovider.NewTestBaseHost(0, hostNode, bridgeInfo, nodetypes.NodeConfig{}, nil),
	}

	emptyBroadcasterHost := Host{
		BaseHost: hostprovider.NewTestBaseHost(
			0,
			node.NewTestNode(nodetypes.NodeConfig{}, db.WithPrefix([]byte("test_host")), nil, nil, nil, nil),
			bridgeInfo,
			nodetypes.NodeConfig{},
			nil,
		),
	}

	cases := []struct {
		name             string
		host             Host
		eventHandlerArgs nodetypes.EventHandlerArgs
		expected         func() (msg string, fields []zapcore.Field)
		err              bool
	}{
		{
			name: "success",
			host: h,
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: hostprovider.RecordBatchEvents(batchSubmitter.GetAddressString()),
			},
			expected: func() (msg string, fields []zapcore.Field) {
				msg = "record batch"
				fields = []zapcore.Field{
					zap.String("submitter", batchSubmitter.GetAddressString()),
				}
				return msg, fields
			},
			err: false,
		},

		{
			name: "different submitter",
			host: h,
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: hostprovider.RecordBatchEvents("another_submitter"),
			},
			expected: nil,
			err:      false,
		},
		{
			name: "empty broadcaster",
			host: emptyBroadcasterHost,
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: hostprovider.RecordBatchEvents(batchSubmitter.GetAddressString()),
			},
			expected: nil,
			err:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logger, observedLogs := logCapturer()
			ctx := types.NewContext(context.Background(), logger, "")

			err := h.recordBatchHandler(ctx, tc.eventHandlerArgs)
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

func TestUpdateBatchInfoHandler(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	hostNode := node.NewTestNode(nodetypes.NodeConfig{}, db.WithPrefix([]byte("test_host")), nil, nil, nil, nil)
	bridgeInfo := ophosttypes.QueryBridgeResponse{
		BridgeId: 1,
	}

	mockBatch := NewMockBatch()

	childCodec, _, err := childprovider.GetCodec("init")
	require.NoError(t, err)
	child := NewMockChild(db.WithPrefix([]byte("test_child")), childCodec, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", 1)
	h := Host{
		BaseHost: hostprovider.NewTestBaseHost(0, hostNode, bridgeInfo, nodetypes.NodeConfig{}, nil),
		batch:    mockBatch,
		child:    child,
	}

	cases := []struct {
		name              string
		host              Host
		eventHandlerArgs  nodetypes.EventHandlerArgs
		expectedBatchInfo *mockBatchInfo
		expectedLog       func() (msg string, fields []zapcore.Field)
		expectedMsg       sdk.Msg
		err               bool
	}{
		{
			name: "success",
			host: h,
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: hostprovider.UpdateBatchInfoEvents(1, ophosttypes.BatchInfo_INITIA, "submitter", 1, 1),
			},
			expectedBatchInfo: &mockBatchInfo{
				chain:         "INITIA",
				submitter:     "submitter",
				outputIndex:   1,
				l2BlockNumber: 1,
			},
			expectedLog: func() (msg string, fields []zapcore.Field) {
				msg = "update batch info"
				fields = []zapcore.Field{
					zap.String("chain", "INITIA"),
					zap.String("submitter", "submitter"),
					zap.Uint64("output_index", 1),
					zap.Int64("l2_block_number", 1),
				}
				return msg, fields
			},
			expectedMsg: opchildtypes.NewMsgSetBridgeInfo("init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", opchildtypes.BridgeInfo{BridgeId: 1, BridgeConfig: ophosttypes.BridgeConfig{BatchInfo: ophosttypes.BatchInfo{Submitter: "submitter", ChainType: ophosttypes.BatchInfo_INITIA}}}),
			err:         false,
		},
		{
			name: "success celestia",
			host: h,
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: hostprovider.UpdateBatchInfoEvents(1, ophosttypes.BatchInfo_CELESTIA, "submitter", 5, 5),
			},
			expectedBatchInfo: &mockBatchInfo{
				chain:         "CELESTIA",
				submitter:     "submitter",
				outputIndex:   5,
				l2BlockNumber: 5,
			},
			expectedLog: func() (msg string, fields []zapcore.Field) {
				msg = "update batch info"
				fields = []zapcore.Field{
					zap.String("chain", "CELESTIA"),
					zap.String("submitter", "submitter"),
					zap.Uint64("output_index", 5),
					zap.Int64("l2_block_number", 5),
				}
				return msg, fields
			},
			expectedMsg: opchildtypes.NewMsgSetBridgeInfo("init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", opchildtypes.BridgeInfo{BridgeId: 1, BridgeConfig: ophosttypes.BridgeConfig{BatchInfo: ophosttypes.BatchInfo{Submitter: "submitter", ChainType: ophosttypes.BatchInfo_CELESTIA}}}),
			err:         false,
		},
		{
			name: "unspecified chain type",
			host: h,
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: hostprovider.UpdateBatchInfoEvents(1, ophosttypes.BatchInfo_UNSPECIFIED, "submitter", 1, 1),
			},
			expectedBatchInfo: nil,
			expectedLog:       nil,
			expectedMsg:       nil,
			err:               true,
		},
		{
			name: "different bridge id",
			host: h,
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: hostprovider.UpdateBatchInfoEvents(2, ophosttypes.BatchInfo_CELESTIA, "submitter", 1, 1),
			},
			expectedBatchInfo: nil,
			expectedLog:       nil,
			expectedMsg:       nil,
			err:               false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logger, observedLogs := logCapturer()
			ctx := types.NewContext(context.Background(), logger, "")

			err := h.updateBatchInfoHandler(ctx, tc.eventHandlerArgs)
			if !tc.err {
				require.NoError(t, err)
				if tc.expectedLog != nil {
					logs := observedLogs.TakeAll()
					require.Len(t, logs, 1)

					expectedMsg, expectedFields := tc.expectedLog()
					require.Equal(t, expectedMsg, logs[0].Message)
					require.Equal(t, expectedFields, logs[0].Context)
				}
				if tc.expectedBatchInfo != nil {
					require.Equal(t, tc.expectedBatchInfo, mockBatch.info)
				}
				msg := h.GetMsgQueue()
				if tc.expectedMsg != nil {
					require.Equal(t, 1, len(msg))
					require.Equal(t, tc.expectedMsg, msg[child.baseAccount][0])
				} else {
					require.Empty(t, msg[child.baseAccount])
				}
			} else {
				require.Error(t, err)
			}
			h.EmptyMsgQueue()
			mockBatch.info = nil
		})
	}
	require.NoError(t, err)
}
