package host

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"testing"

	abcitypes "github.com/cometbft/cometbft/abci/types"
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

func ProposeOutputEvents(
	proposer string,
	bridgeId uint64,
	outputIndex uint64,
	l2BlockNumber uint64,
	outputRoot []byte,
) []abcitypes.EventAttribute {
	return []abcitypes.EventAttribute{
		{
			Key:   ophosttypes.AttributeKeyProposer,
			Value: proposer,
		},
		{
			Key:   ophosttypes.AttributeKeyBridgeId,
			Value: strconv.FormatUint(bridgeId, 10),
		},
		{
			Key:   ophosttypes.AttributeKeyOutputIndex,
			Value: strconv.FormatUint(outputIndex, 10),
		},
		{
			Key:   ophosttypes.AttributeKeyL2BlockNumber,
			Value: strconv.FormatUint(l2BlockNumber, 10),
		},
		{
			Key:   ophosttypes.AttributeKeyOutputRoot,
			Value: hex.EncodeToString(outputRoot),
		},
	}
}

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

	fullAttributes := ProposeOutputEvents("proposer", 1, 2, 3, []byte("output_root"))

	cases := []struct {
		name             string
		eventHandlerArgs nodetypes.EventHandlerArgs
		expected         func() (msg string, fields []zapcore.Field)
		err              bool
	}{
		{
			name: "success",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: ProposeOutputEvents("proposer", 1, 2, 3, []byte("output_root")),
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
				EventAttributes: ProposeOutputEvents("proposer", 2, 2, 3, []byte("output_root")),
			},
			expected: nil,
			err:      false,
		},
		{
			name: "missing event attribute proposer",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: fullAttributes[1:],
			},
			expected: nil,
			err:      true,
		},
		{
			name: "missing event attribute bridge id",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: append(fullAttributes[:1], fullAttributes[2:]...),
			},
			expected: nil,
			err:      true,
		},
		{
			name: "missing event attribute output index",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: append(fullAttributes[:2], fullAttributes[3:]...),
			},
			expected: nil,
			err:      true,
		},
		{
			name: "missing event attribute l2 block number",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: append(fullAttributes[:3], fullAttributes[4:]...),
			},
			expected: nil,
			err:      true,
		},
		{
			name: "missing event attribute output root",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: fullAttributes[:4],
			},
			expected: nil,
			err:      true,
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

func FinalizeWithdrawalEvents(
	bridgeId uint64,
	outputIndex uint64,
	l2Sequence uint64,
	from string,
	to string,
	l1Denom string,
	l2Denom string,
	amount sdk.Coin,
) []abcitypes.EventAttribute {
	return []abcitypes.EventAttribute{
		{
			Key:   ophosttypes.AttributeKeyBridgeId,
			Value: strconv.FormatUint(bridgeId, 10),
		},
		{
			Key:   ophosttypes.AttributeKeyOutputIndex,
			Value: strconv.FormatUint(outputIndex, 10),
		},
		{
			Key:   ophosttypes.AttributeKeyL2Sequence,
			Value: strconv.FormatUint(l2Sequence, 10),
		},
		{
			Key:   ophosttypes.AttributeKeyFrom,
			Value: from,
		},
		{
			Key:   ophosttypes.AttributeKeyTo,
			Value: to,
		},
		{
			Key:   ophosttypes.AttributeKeyL1Denom,
			Value: l1Denom,
		},
		{
			Key:   ophosttypes.AttributeKeyL2Denom,
			Value: l2Denom,
		},
		{
			Key:   ophosttypes.AttributeKeyAmount,
			Value: amount.String(),
		},
	}
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

	fullAttributes := FinalizeWithdrawalEvents(1, 2, 3, "from", "to", "l1Denom", "l2Denom", sdk.NewInt64Coin("uinit", 10000))

	cases := []struct {
		name             string
		eventHandlerArgs nodetypes.EventHandlerArgs
		expected         func() (msg string, fields []zapcore.Field)
		err              bool
	}{
		{
			name: "success",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: FinalizeWithdrawalEvents(1, 2, 3, "from", "to", "l1Denom", "l2Denom", sdk.NewInt64Coin("uinit", 10000)),
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
				EventAttributes: FinalizeWithdrawalEvents(2, 2, 3, "from", "to", "l1Denom", "l2Denom", sdk.NewInt64Coin("uinit", 10000)),
			},
			expected: nil,
			err:      false,
		},
		{
			name: "missing event attribute bridge id",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: fullAttributes[1:],
			},
			expected: nil,
			err:      true,
		},
		{
			name: "missing event attribute output index",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: append(fullAttributes[:1], fullAttributes[2:]...),
			},
			expected: nil,
			err:      true,
		},
		{
			name: "missing event attribute l2 sequence",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: append(fullAttributes[:2], fullAttributes[3:]...),
			},
			expected: nil,
			err:      true,
		},
		{
			name: "missing event attribute from",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: append(fullAttributes[:3], fullAttributes[4:]...),
			},
			expected: nil,
			err:      true,
		},
		{
			name: "missing event attribute to",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: append(fullAttributes[:4], fullAttributes[5:]...),
			},
			expected: nil,
			err:      true,
		},
		{
			name: "missing event attribute l1 denom",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: append(fullAttributes[:5], fullAttributes[6:]...),
			},
			expected: nil,
			err:      true,
		},
		{
			name: "missing event attribute l2 denom",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: append(fullAttributes[:6], fullAttributes[7:]...),
			},
			expected: nil,
			err:      true,
		},
		{
			name: "missing event attribute amount",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: fullAttributes[:7],
			},
			expected: nil,
			err:      true,
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
