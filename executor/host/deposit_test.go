package host

import (
	"testing"
	"time"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/node"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

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
				EventAttributes: hostprovider.InitiateTokenDepositEvents(1, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l1Denom", 100), []byte("databytes"), 1, "l2denom"),
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
				EventAttributes: hostprovider.InitiateTokenDepositEvents(2, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l1Denom", 100), []byte("databytes"), 1, "l2denom"),
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
				EventAttributes: hostprovider.InitiateTokenDepositEvents(1, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l1Denom", 100), []byte("databytes"), 1, "l2denom"),
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
				EventAttributes: hostprovider.InitiateTokenDepositEvents(1, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", sdk.NewInt64Coin("l1Denom", 100), []byte("databytes"), 1, "l2denom"),
			},
			expected: nil,
			err:      false,
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
