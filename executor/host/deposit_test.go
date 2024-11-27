package host

import (
	"encoding/hex"
	"strconv"
	"testing"
	"time"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/node"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/stretchr/testify/require"
)

func InitiateTokenDepositEvents(
	bridgeId uint64,
	sender, to string,
	amount sdk.Coin,
	data []byte,
	l1Sequence uint64,
	l2Denom string,
) []abcitypes.EventAttribute {
	return []abcitypes.EventAttribute{
		{
			Key:   ophosttypes.AttributeKeyBridgeId,
			Value: strconv.FormatUint(bridgeId, 10),
		},
		{
			Key:   ophosttypes.AttributeKeyL1Sequence,
			Value: strconv.FormatUint(l1Sequence, 10),
		},
		{
			Key:   ophosttypes.AttributeKeyFrom,
			Value: sender,
		},
		{
			Key:   ophosttypes.AttributeKeyTo,
			Value: to,
		},
		{
			Key:   ophosttypes.AttributeKeyL1Denom,
			Value: amount.Denom,
		},
		{
			Key:   ophosttypes.AttributeKeyL2Denom,
			Value: l2Denom,
		},
		{
			Key:   ophosttypes.AttributeKeyAmount,
			Value: amount.Amount.String(),
		},
		{
			Key:   ophosttypes.AttributeKeyData,
			Value: hex.EncodeToString(data),
		},
	}
}

func TestInitializeDepositHandler(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)
	bridgeInfo := ophosttypes.QueryBridgeResponse{
		BridgeId: 1,
	}
	hostNode := node.NewTestNode(nodetypes.NodeConfig{}, db.WithPrefix([]byte("test_host")), nil, nil, nil, nil)
	childCodec, _, err := childprovider.GetCodec("init")
	h := Host{
		BaseHost: hostprovider.NewTestBaseHost(0, hostNode, bridgeInfo, nodetypes.NodeConfig{}, nil),
	}

	fullAttributes := InitiateTokenDepositEvents(1, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l1Denom", 100), []byte("databytes"), 1, "l2denom")

	cases := []struct {
		name              string
		initialL1Sequence uint64
		child             *mockChild
		eventHandlerArgs  nodetypes.EventHandlerArgs
		expected          sdk.Msg
		err               bool
	}{
		{
			name:              "success",
			initialL1Sequence: 0,
			child:             NewMockChild(db.WithPrefix([]byte("test_child")), childCodec, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "", 1),
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				BlockHeight:     1,
				BlockTime:       time.Time{},
				LatestHeight:    1,
				TxIndex:         0,
				EventAttributes: InitiateTokenDepositEvents(1, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l1Denom", 100), []byte("databytes"), 1, "l2denom"),
			},
			expected: opchildtypes.NewMsgFinalizeTokenDeposit("init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l2denom", 100), 1, 1, "l1Denom", []byte("databytes")),
			err:      false,
		},
		{
			name:              "another bridge id",
			initialL1Sequence: 0,
			child:             NewMockChild(db.WithPrefix([]byte("test_child")), childCodec, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "", 1),
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				BlockHeight:     1,
				BlockTime:       time.Time{},
				LatestHeight:    1,
				TxIndex:         0,
				EventAttributes: InitiateTokenDepositEvents(2, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l1Denom", 100), []byte("databytes"), 1, "l2denom"),
			},
			expected: nil,
			err:      false,
		},
		{
			name:              "empty child broadcaster",
			initialL1Sequence: 0,
			child:             NewMockChild(db.WithPrefix([]byte("test_child")), childCodec, "", "", 1),
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				BlockHeight:     1,
				BlockTime:       time.Time{},
				LatestHeight:    1,
				TxIndex:         0,
				EventAttributes: InitiateTokenDepositEvents(2, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l1Denom", 100), []byte("databytes"), 1, "l2denom"),
			},
			expected: nil,
			err:      false,
		},
		{
			name:              "processed l1 sequence",
			initialL1Sequence: 2,
			child:             NewMockChild(db.WithPrefix([]byte("test_child")), childCodec, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "", 1),
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				BlockHeight:     1,
				BlockTime:       time.Time{},
				LatestHeight:    1,
				TxIndex:         0,
				EventAttributes: InitiateTokenDepositEvents(2, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l1Denom", 100), []byte("databytes"), 1, "l2denom"),
			},
			expected: nil,
			err:      false,
		},
		{
			name:              "missing event attribute bridge id",
			initialL1Sequence: 0,
			child:             NewMockChild(db.WithPrefix([]byte("test_child")), childCodec, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "", 1),
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				BlockHeight:     1,
				BlockTime:       time.Time{},
				LatestHeight:    1,
				TxIndex:         0,
				EventAttributes: fullAttributes[1:],
			},
			expected: opchildtypes.NewMsgFinalizeTokenDeposit("init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l2denom", 100), 1, 1, "l1Denom", []byte("databytes")),
			err:      true,
		},
		{
			name:              "missing event attribute l1 sequence",
			initialL1Sequence: 0,
			child:             NewMockChild(db.WithPrefix([]byte("test_child")), childCodec, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "", 1),
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				BlockHeight:     1,
				BlockTime:       time.Time{},
				LatestHeight:    1,
				TxIndex:         0,
				EventAttributes: append(fullAttributes[:1], fullAttributes[2:]...),
			},
			expected: opchildtypes.NewMsgFinalizeTokenDeposit("init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l2denom", 100), 1, 1, "l1Denom", []byte("databytes")),
			err:      true,
		},
		{
			name:              "missing event attribute from",
			initialL1Sequence: 0,
			child:             NewMockChild(db.WithPrefix([]byte("test_child")), childCodec, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "", 1),
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				BlockHeight:     1,
				BlockTime:       time.Time{},
				LatestHeight:    1,
				TxIndex:         0,
				EventAttributes: append(fullAttributes[:2], fullAttributes[3:]...),
			},
			expected: opchildtypes.NewMsgFinalizeTokenDeposit("init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l2denom", 100), 1, 1, "l1Denom", []byte("databytes")),
			err:      true,
		},
		{
			name:              "missing event attribute to",
			initialL1Sequence: 0,
			child:             NewMockChild(db.WithPrefix([]byte("test_child")), childCodec, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "", 1),
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				BlockHeight:     1,
				BlockTime:       time.Time{},
				LatestHeight:    1,
				TxIndex:         0,
				EventAttributes: append(fullAttributes[:3], fullAttributes[4:]...),
			},
			expected: opchildtypes.NewMsgFinalizeTokenDeposit("init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l2denom", 100), 1, 1, "l1Denom", []byte("databytes")),
			err:      true,
		},
		{
			name:              "missing event attribute l1 denom",
			initialL1Sequence: 0,
			child:             NewMockChild(db.WithPrefix([]byte("test_child")), childCodec, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "", 1),
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				BlockHeight:     1,
				BlockTime:       time.Time{},
				LatestHeight:    1,
				TxIndex:         0,
				EventAttributes: append(fullAttributes[:4], fullAttributes[5:]...),
			},
			expected: opchildtypes.NewMsgFinalizeTokenDeposit("init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l2denom", 100), 1, 1, "l1Denom", []byte("databytes")),
			err:      true,
		},
		{
			name:              "missing event attribute l2 denom",
			initialL1Sequence: 0,
			child:             NewMockChild(db.WithPrefix([]byte("test_child")), childCodec, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "", 1),
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				BlockHeight:     1,
				BlockTime:       time.Time{},
				LatestHeight:    1,
				TxIndex:         0,
				EventAttributes: append(fullAttributes[:5], fullAttributes[6:]...),
			},
			expected: opchildtypes.NewMsgFinalizeTokenDeposit("init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l2denom", 100), 1, 1, "l1Denom", []byte("databytes")),
			err:      true,
		},
		{
			name:              "missing event attribute amount",
			initialL1Sequence: 0,
			child:             NewMockChild(db.WithPrefix([]byte("test_child")), childCodec, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "", 1),
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				BlockHeight:     1,
				BlockTime:       time.Time{},
				LatestHeight:    1,
				TxIndex:         0,
				EventAttributes: append(fullAttributes[:6], fullAttributes[7:]...),
			},
			expected: opchildtypes.NewMsgFinalizeTokenDeposit("init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l2denom", 100), 1, 1, "l1Denom", []byte("databytes")),
			err:      true,
		},
		{
			name:              "missing event attribute data",
			initialL1Sequence: 0,
			child:             NewMockChild(db.WithPrefix([]byte("test_child")), childCodec, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "", 1),
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				BlockHeight:     1,
				BlockTime:       time.Time{},
				LatestHeight:    1,
				TxIndex:         0,
				EventAttributes: fullAttributes[:7],
			},
			expected: opchildtypes.NewMsgFinalizeTokenDeposit("init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l2denom", 100), 1, 1, "l1Denom", []byte("databytes")),
			err:      true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h.initialL1Sequence = tc.initialL1Sequence
			h.child = tc.child

			err := h.initiateDepositHandler(types.Context{}, tc.eventHandlerArgs)
			if !tc.err {
				require.NoError(t, err)
				msg := h.GetMsgQueue()
				if tc.expected != nil {
					require.Equal(t, 1, len(msg))
					require.Equal(t, tc.expected, msg[tc.child.baseAccount][0])
				} else {
					require.Empty(t, msg[tc.child.baseAccount])
				}
			} else {
				require.Error(t, err)
			}
			h.EmptyMsgQueue()
		})
	}
	require.NoError(t, err)
}
