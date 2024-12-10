package host

import (
	"context"
	"strconv"
	"testing"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/node"
	"github.com/initia-labs/opinit-bots/node/broadcaster"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func RecordBatchEvents(
	submitter string,
) []abcitypes.EventAttribute {
	return []abcitypes.EventAttribute{
		{
			Key:   ophosttypes.AttributeKeySubmitter,
			Value: submitter,
		},
	}
}

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
				EventAttributes: RecordBatchEvents(batchSubmitter.GetAddressString()),
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
				EventAttributes: RecordBatchEvents("another_submitter"),
			},
			expected: nil,
			err:      false,
		},
		{
			name: "empty broadcaster",
			host: emptyBroadcasterHost,
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: RecordBatchEvents(batchSubmitter.GetAddressString()),
			},
			expected: nil,
			err:      false,
		},
		{
			name: "missing event attribute submitter",
			host: emptyBroadcasterHost,
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: RecordBatchEvents(batchSubmitter.GetAddressString())[1:],
			},
			expected: nil,
			err:      true,
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

func UpdateBatchInfoEvents(
	bridgeId uint64,
	chainType ophosttypes.BatchInfo_ChainType,
	submitter string,
	finalizedOutputIndex uint64,
	l2BlockNumber uint64,
) []abcitypes.EventAttribute {
	return []abcitypes.EventAttribute{
		{
			Key:   ophosttypes.AttributeKeyBridgeId,
			Value: strconv.FormatUint(bridgeId, 10),
		},
		{
			Key:   ophosttypes.AttributeKeyBatchChainType,
			Value: chainType.StringWithoutPrefix(),
		},
		{
			Key:   ophosttypes.AttributeKeyBatchSubmitter,
			Value: submitter,
		},
		{
			Key:   ophosttypes.AttributeKeyFinalizedOutputIndex,
			Value: strconv.FormatUint(finalizedOutputIndex, 10),
		},
		{
			Key:   ophosttypes.AttributeKeyFinalizedL2BlockNumber,
			Value: strconv.FormatUint(l2BlockNumber, 10),
		},
	}
}

func TestUpdateBatchInfoHandler(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	hostNode := node.NewTestNode(nodetypes.NodeConfig{}, db.WithPrefix([]byte("test_host")), nil, nil, nil, nil)
	bridgeInfo := ophosttypes.QueryBridgeResponse{
		BridgeId: 1,
	}

	mockBatch := NewMockBatch()
	h := Host{
		BaseHost: hostprovider.NewTestBaseHost(0, hostNode, bridgeInfo, nodetypes.NodeConfig{}, nil),
		batch:    mockBatch,
	}

	fullAttributes := UpdateBatchInfoEvents(1, ophosttypes.BatchInfo_CHAIN_TYPE_INITIA, "submitter", 1, 1)

	cases := []struct {
		name              string
		host              Host
		eventHandlerArgs  nodetypes.EventHandlerArgs
		expectedBatchInfo *mockBatchInfo
		expectedLog       func() (msg string, fields []zapcore.Field)
		err               bool
	}{
		{
			name: "success",
			host: h,
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: UpdateBatchInfoEvents(1, ophosttypes.BatchInfo_CHAIN_TYPE_INITIA, "submitter", 1, 1),
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
			err: false,
		},
		{
			name: "unspecified chain type",
			host: h,
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: UpdateBatchInfoEvents(1, ophosttypes.BatchInfo_CHAIN_TYPE_UNSPECIFIED, "submitter", 1, 1),
			},
			expectedBatchInfo: nil,
			expectedLog:       nil,
			err:               true,
		},
		{
			name: "different bridge id",
			host: h,
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: UpdateBatchInfoEvents(2, ophosttypes.BatchInfo_CHAIN_TYPE_CELESTIA, "submitter", 1, 1),
			},
			expectedBatchInfo: nil,
			expectedLog:       nil,
			err:               false,
		},
		{
			name: "missing event attribute bridge id",
			host: h,
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: fullAttributes[1:],
			},
			expectedBatchInfo: nil,
			expectedLog:       nil,
			err:               true,
		},
		{
			name: "missing event attribute batch chain type",
			host: h,
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: append(fullAttributes[:1], fullAttributes[2:]...),
			},
			expectedBatchInfo: nil,
			expectedLog:       nil,
			err:               true,
		},
		{
			name: "missing event attribute submitter",
			host: h,
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: append(fullAttributes[:2], fullAttributes[3:]...),
			},
			expectedBatchInfo: nil,
			expectedLog:       nil,
			err:               true,
		},
		{
			name: "missing event attribute output index",
			host: h,
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: append(fullAttributes[:3], fullAttributes[4:]...),
			},
			expectedBatchInfo: nil,
			expectedLog:       nil,
			err:               true,
		},
		{
			name: "missing event attribute l2 block number",
			host: h,
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: fullAttributes[:4],
			},
			expectedBatchInfo: nil,
			expectedLog:       nil,
			err:               true,
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
			} else {
				require.Error(t, err)
			}
			mockBatch.info = nil
		})
	}
	require.NoError(t, err)
}
