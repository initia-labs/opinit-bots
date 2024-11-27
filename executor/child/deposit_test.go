package child

import (
	"context"
	"strconv"
	"testing"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/node"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func FinalizeDepositEvents(
	l1Sequence uint64,
	sender string,
	recipient string,
	denom string,
	baseDenom string,
	amount sdk.Coin,
	finalizeHeight uint64,
) []abcitypes.EventAttribute {
	return []abcitypes.EventAttribute{
		{
			Key:   opchildtypes.AttributeKeyL1Sequence,
			Value: strconv.FormatUint(l1Sequence, 10),
		},
		{
			Key:   opchildtypes.AttributeKeySender,
			Value: sender,
		},
		{
			Key:   opchildtypes.AttributeKeyRecipient,
			Value: recipient,
		},
		{
			Key:   opchildtypes.AttributeKeyDenom,
			Value: denom,
		},
		{
			Key:   opchildtypes.AttributeKeyBaseDenom,
			Value: baseDenom,
		},
		{
			Key:   opchildtypes.AttributeKeyAmount,
			Value: amount.Amount.String(),
		},
		{
			Key:   opchildtypes.AttributeKeyFinalizeHeight,
			Value: strconv.FormatUint(finalizeHeight, 10),
		},
	}
}

func TestFinalizeDepositHandler(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	childNode := node.NewTestNode(nodetypes.NodeConfig{}, db.WithPrefix([]byte("test_child")), nil, nil, nil, nil)
	bridgeInfo := ophosttypes.QueryBridgeResponse{
		BridgeId: 1,
	}

	ch := Child{
		BaseChild: childprovider.NewTestBaseChild(0, childNode, nil, bridgeInfo, nil, nodetypes.NodeConfig{}),
	}

	fullAttributes := FinalizeDepositEvents(1, "sender", "recipient", "denom", "baseDenom", sdk.NewInt64Coin("uinit", 10000), 2)

	cases := []struct {
		name             string
		eventHandlerArgs nodetypes.EventHandlerArgs
		expected         func() (msg string, fields []zapcore.Field)
		err              bool
	}{
		{
			name: "success",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: FinalizeDepositEvents(1, "sender", "recipient", "denom", "baseDenom", sdk.NewInt64Coin("uinit", 10000), 2),
			},
			expected: func() (msg string, fields []zapcore.Field) {
				msg = "finalize token deposit"
				fields = []zapcore.Field{
					zap.Int64("l1_blockHeight", 2),
					zap.Uint64("l1_sequence", 1),
					zap.String("from", "sender"),
					zap.String("to", "recipient"),
					zap.String("amount", "10000denom"),
					zap.String("base_denom", "baseDenom"),
				}
				return msg, fields
			},
			err: false,
		},
		{
			name: "missing event attribute l1 sequence",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: fullAttributes[1:],
			},
			expected: nil,
			err:      true,
		},
		{
			name: "missing event attribute sender",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: append(fullAttributes[:1], fullAttributes[2:]...),
			},
			expected: nil,
			err:      true,
		},
		{
			name: "missing event attribute recipient",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: append(fullAttributes[:2], fullAttributes[3:]...),
			},
			expected: nil,
			err:      true,
		},
		{
			name: "missing event attribute l1 denom",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: append(fullAttributes[:3], fullAttributes[4:]...),
			},
			expected: nil,
			err:      true,
		},
		{
			name: "missing event attribute l2 denom",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: append(fullAttributes[:4], fullAttributes[5:]...),
			},
			expected: nil,
			err:      true,
		},
		{
			name: "missing event attribute amount",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: append(fullAttributes[:5], fullAttributes[6:]...),
			},
			expected: nil,
			err:      true,
		},
		{
			name: "missing event attribute finalize height",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: fullAttributes[:6],
			},
			expected: nil,
			err:      true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logger, observedLogs := logCapturer()
			ctx := types.NewContext(context.Background(), logger, "")

			err := ch.finalizeDepositHandler(ctx, tc.eventHandlerArgs)
			if !tc.err {
				require.NoError(t, err)
				if tc.expected != nil {
					logs := observedLogs.TakeAll()
					require.Len(t, logs, 1)

					expectedMsg, expectedFields := tc.expected()
					require.Equal(t, expectedMsg, logs[0].Message)
					require.Equal(t, expectedFields, logs[0].Context)

					expectedL1Height, err := strconv.ParseInt(tc.eventHandlerArgs.EventAttributes[6].Value, 10, 64)
					require.NoError(t, err)
					expectedL1Sequence, err := strconv.ParseUint(tc.eventHandlerArgs.EventAttributes[0].Value, 10, 64)
					require.NoError(t, err)

					require.Equal(t, expectedL1Height, ch.lastFinalizedDepositL1BlockHeight)
					require.Equal(t, expectedL1Sequence, ch.lastFinalizedDepositL1Sequence)
				}
			} else {
				require.Error(t, err)
			}
		})
	}
	require.NoError(t, err)
}
