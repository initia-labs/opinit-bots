package host

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/node"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"github.com/initia-labs/opinit-bots/types"
)

func TestUpdateProposerHandler(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)
	hostNode := node.NewTestNode(nodetypes.NodeConfig{}, db.WithPrefix([]byte("test_host")), nil, nil, nil, nil)
	h := Host{
		BaseHost: hostprovider.NewTestBaseHost(0, hostNode, ophosttypes.QueryBridgeResponse{
			BridgeId: 1,
			BridgeConfig: ophosttypes.BridgeConfig{
				Proposer: "proposer",
			},
		}, nodetypes.NodeConfig{}, nil),
	}

	cases := []struct {
		name                   string
		bridgeId               uint64
		proposer               string
		finalizedOutputIndex   uint64
		finalizedL2BlockNumber uint64
		expectedProposer       string
		expectedLog            func() (msg string, fields []zapcore.Field)
		err                    bool
	}{
		{
			name:                   "success",
			bridgeId:               1,
			proposer:               "proposer",
			finalizedOutputIndex:   1,
			finalizedL2BlockNumber: 1,
			expectedProposer:       "proposer",
			expectedLog: func() (msg string, fields []zapcore.Field) {
				msg = "update proposer"
				fields = []zapcore.Field{
					zap.Uint64("bridge_id", 1),
					zap.String("proposer", "proposer"),
					zap.Uint64("finalized_output_index", 1),
					zap.Uint64("finalized_l2_block_number", 1),
				}
				return msg, fields
			},
		},
		{
			name:                   "another bridge id",
			bridgeId:               2,
			proposer:               "proposer",
			finalizedOutputIndex:   5,
			finalizedL2BlockNumber: 5,
			expectedProposer:       "proposer",
			expectedLog:            nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logger, observedLogs := logCapturer()
			ctx := types.NewContext(context.Background(), logger, "")

			err := h.updateProposerHandler(ctx, nodetypes.EventHandlerArgs{
				EventAttributes: hostprovider.UpdateProposerEvents(tc.bridgeId, tc.proposer, tc.finalizedOutputIndex, tc.finalizedL2BlockNumber),
			})
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedProposer, h.BridgeInfo().BridgeConfig.Proposer)
				if tc.expectedLog != nil {
					logs := observedLogs.TakeAll()
					require.Len(t, logs, 1)

					expectedMsg, expectedFields := tc.expectedLog()
					require.Equal(t, expectedMsg, logs[0].Message)
					require.Equal(t, expectedFields, logs[0].Context)
				}
			}
		})
	}
}

func TestUpdateChallengerHandler(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)
	hostNode := node.NewTestNode(nodetypes.NodeConfig{}, db.WithPrefix([]byte("test_host")), nil, nil, nil, nil)
	h := Host{
		BaseHost: hostprovider.NewTestBaseHost(0, hostNode, ophosttypes.QueryBridgeResponse{
			BridgeId: 1,
			BridgeConfig: ophosttypes.BridgeConfig{
				Challenger: "challenger",
			},
		}, nodetypes.NodeConfig{}, nil),
	}

	cases := []struct {
		name                   string
		bridgeId               uint64
		challenger             string
		finalizedOutputIndex   uint64
		finalizedL2BlockNumber uint64
		expectedChallenger     string
		expectedLog            func() (msg string, fields []zapcore.Field)
		err                    bool
	}{
		{
			name:                   "success",
			bridgeId:               1,
			challenger:             "challenger",
			finalizedOutputIndex:   1,
			finalizedL2BlockNumber: 1,
			expectedChallenger:     "challenger",
			expectedLog: func() (msg string, fields []zapcore.Field) {
				msg = "update challenger"
				fields = []zapcore.Field{
					zap.Uint64("bridge_id", 1),
					zap.String("challenger", "challenger"),
					zap.Uint64("finalized_output_index", 1),
					zap.Uint64("finalized_l2_block_number", 1),
				}
				return msg, fields
			},
		},
		{
			name:                   "another bridge id",
			bridgeId:               2,
			challenger:             "challenger",
			finalizedOutputIndex:   5,
			finalizedL2BlockNumber: 5,
			expectedChallenger:     "challenger",
			expectedLog:            nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logger, observedLogs := logCapturer()
			ctx := types.NewContext(context.Background(), logger, "")

			err := h.updateChallengerHandler(ctx, nodetypes.EventHandlerArgs{
				EventAttributes: hostprovider.UpdateChallengerEvents(tc.bridgeId, tc.challenger, tc.finalizedOutputIndex, tc.finalizedL2BlockNumber),
			})
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedChallenger, h.BridgeInfo().BridgeConfig.Challenger)
				if tc.expectedLog != nil {
					logs := observedLogs.TakeAll()
					require.Len(t, logs, 1)

					expectedMsg, expectedFields := tc.expectedLog()
					require.Equal(t, expectedMsg, logs[0].Message)
					require.Equal(t, expectedFields, logs[0].Context)
				}
			}
		})
	}
}
