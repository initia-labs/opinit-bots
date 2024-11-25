package host

import (
	"testing"
	"time"

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

func TestTxHandler(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)
	hostNode := node.NewTestNode(nodetypes.NodeConfig{}, db.WithPrefix([]byte("test_host")), nil, nil, nil, nil)
	childCodec, _, err := childprovider.GetCodec("init")

	cases := []struct {
		name          string
		oracleEnabled bool
		child         *mockChild
		txHandlerArgs nodetypes.TxHandlerArgs
		expected      func() (sender string, msg sdk.Msg, err error)
		err           bool
	}{
		{
			name:          "success",
			oracleEnabled: true,
			child:         NewMockChild(db.WithPrefix([]byte("test_child")), childCodec, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", 1),
			txHandlerArgs: nodetypes.TxHandlerArgs{
				BlockHeight:  1,
				BlockTime:    time.Time{},
				LatestHeight: 1,
				TxIndex:      0,
				Tx:           []byte("oracle_tx"),
			},
			expected: func() (sender string, msg sdk.Msg, err error) {
				msg, err = childprovider.CreateAuthzMsg("init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", opchildtypes.NewMsgUpdateOracle("init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", 1, []byte("oracle_tx")))
				sender = "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5"
				return sender, msg, err
			},
			err: false,
		},
		{
			name:          "empty tx",
			oracleEnabled: true,
			child:         NewMockChild(db.WithPrefix([]byte("test_child")), childCodec, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", 1),
			txHandlerArgs: nodetypes.TxHandlerArgs{
				BlockHeight:  1,
				BlockTime:    time.Time{},
				LatestHeight: 1,
				TxIndex:      0,
				Tx:           []byte(""),
			},
			expected: func() (sender string, msg sdk.Msg, err error) {
				msg, err = childprovider.CreateAuthzMsg("init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", opchildtypes.NewMsgUpdateOracle("init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", 1, []byte("")))
				sender = "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5"
				return sender, msg, err
			},
			err: false,
		},
		{
			name:          "old height",
			oracleEnabled: true,
			child:         NewMockChild(db.WithPrefix([]byte("test_child")), childCodec, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", 1),
			txHandlerArgs: nodetypes.TxHandlerArgs{
				BlockHeight:  1,
				BlockTime:    time.Time{},
				LatestHeight: 3,
				TxIndex:      0,
				Tx:           []byte(""),
			},
			expected: nil,
			err:      false,
		},
		{
			name:          "another tx",
			oracleEnabled: true,
			child:         NewMockChild(db.WithPrefix([]byte("test_child")), childCodec, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", 1),
			txHandlerArgs: nodetypes.TxHandlerArgs{
				BlockHeight:  1,
				BlockTime:    time.Time{},
				LatestHeight: 1,
				TxIndex:      1,
				Tx:           []byte(""),
			},
			expected: nil,
			err:      false,
		},
		{
			name:          "oracle disabled",
			oracleEnabled: false,
			child:         NewMockChild(db.WithPrefix([]byte("test_child")), childCodec, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", 1),
			txHandlerArgs: nodetypes.TxHandlerArgs{
				BlockHeight:  1,
				BlockTime:    time.Time{},
				LatestHeight: 1,
				TxIndex:      0,
				Tx:           []byte(""),
			},
			expected: nil,
			err:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := Host{
				BaseHost: hostprovider.NewTestBaseHost(0, hostNode, ophosttypes.QueryBridgeResponse{
					BridgeId: 1,
					BridgeConfig: ophosttypes.BridgeConfig{
						OracleEnabled: tc.oracleEnabled,
					},
				}, nodetypes.NodeConfig{}, nil),
				child: tc.child,
			}

			err := h.txHandler(types.Context{}, tc.txHandlerArgs)
			if !tc.err {
				require.NoError(t, err)
				processedMsgs := h.GetProcessedMsgs()
				if tc.expected != nil {
					require.Equal(t, 1, len(processedMsgs))
					expectedSender, expectedMsg, err := tc.expected()
					require.NoError(t, err)
					require.Equal(t, expectedSender, processedMsgs[0].Sender)
					require.Equal(t, expectedMsg, processedMsgs[0].Msgs[0])
				} else {
					require.Empty(t, processedMsgs)
				}
			} else {
				require.Error(t, err)
			}
			h.EmptyProcessedMsgs()
		})
	}
	require.NoError(t, err)
}
