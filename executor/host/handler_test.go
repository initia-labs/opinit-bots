package host

import (
	"testing"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/node"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/stretchr/testify/require"
)

func TestBeginBlockHandler(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)
	hostNode := node.NewTestNode(nodetypes.NodeConfig{}, db.WithPrefix([]byte("test_host")), nil, nil, nil, nil)

	h := Host{
		BaseHost: hostprovider.NewTestBaseHost(0, hostNode, ophosttypes.QueryBridgeResponse{
			BridgeId: 1,
		}, nodetypes.NodeConfig{}, nil),
		stage: db.NewStage(),
	}

	msgQueue := h.GetMsgQueue()
	require.Empty(t, msgQueue["init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5"])
	h.AppendMsgQueue(opchildtypes.NewMsgUpdateOracle("init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", 1, []byte("oracle_tx")), "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5")
	msgQueue = h.GetMsgQueue()
	require.Len(t, msgQueue["init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5"], 1)

	msgs := h.GetProcessedMsgs()
	require.Empty(t, msgs)
	h.AppendProcessedMsgs(btypes.ProcessedMsgs{})
	msgs = h.GetProcessedMsgs()
	require.Len(t, msgs, 1)

	require.Equal(t, 0, h.stage.Len())
	err = h.stage.Set([]byte("key"), []byte("value"))
	require.NoError(t, err)
	require.Equal(t, 1, h.stage.Len())

	err = h.beginBlockHandler(types.Context{}, nodetypes.BeginBlockArgs{})
	require.NoError(t, err)

	msgQueue = h.GetMsgQueue()
	require.Empty(t, msgQueue["init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5"])

	msgs = h.GetProcessedMsgs()
	require.Empty(t, msgs)

	require.Equal(t, 0, h.stage.Len())
}

func TestEndBlockHandler(t *testing.T) {
	childCodec, _, err := childprovider.GetCodec("init")

	mockCount := int64(0)
	mockTimestampFetcher := func() int64 {
		mockCount++
		return mockCount
	}
	types.CurrentNanoTimestamp = mockTimestampFetcher

	cases := []struct {
		name                  string
		child                 *mockChild
		msgQueue              map[string][]sdk.Msg
		processedMsgs         []btypes.ProcessedMsgs
		dbChanges             []types.KV
		endBlockArgs          nodetypes.EndBlockArgs
		expectedProcessedMsgs []btypes.ProcessedMsgs
		expectedDB            []types.KV
		err                   bool
	}{
		{
			name:  "success",
			child: NewMockChild(nil, childCodec, "sender0", "sender1", 1),
			msgQueue: map[string][]sdk.Msg{
				"sender0": {&opchildtypes.MsgFinalizeTokenDeposit{}},
			},
			processedMsgs: []btypes.ProcessedMsgs{
				{
					Sender:    "sender1",
					Msgs:      []sdk.Msg{&opchildtypes.MsgUpdateOracle{}},
					Timestamp: 10000,
					Save:      true,
				},
			},
			dbChanges: []types.KV{
				{
					Key:   []byte("key1"),
					Value: []byte("value1"),
				},
			},
			endBlockArgs: nodetypes.EndBlockArgs{
				Block: cmtproto.Block{
					Header: cmtproto.Header{
						Height: 10,
					},
				},
			},
			expectedProcessedMsgs: []btypes.ProcessedMsgs{
				{
					Sender:    "sender1",
					Msgs:      []sdk.Msg{&opchildtypes.MsgUpdateOracle{}},
					Timestamp: 10000,
					Save:      true,
				},
				{
					Sender:    "sender0",
					Msgs:      []sdk.Msg{&opchildtypes.MsgFinalizeTokenDeposit{}},
					Timestamp: 1,
					Save:      true,
				},
			},
			expectedDB: []types.KV{
				{
					Key:   []byte("test_host/key1"),
					Value: []byte("value1"),
				},
				{
					Key:   []byte("test_host/synced_height"),
					Value: []byte("10"),
				},
				{
					Key:   append([]byte("test_child/processed_msgs/"), []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x27, 0x10}...),
					Value: []byte(`{"sender":"sender1","msgs":["{\"@type\":\"/opinit.opchild.v1.MsgUpdateOracle\",\"sender\":\"\",\"height\":\"0\",\"data\":null}"],"timestamp":10000,"save":true}`),
				},
				{
					Key:   append([]byte("test_child/processed_msgs/"), []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1}...),
					Value: []byte(`{"sender":"sender0","msgs":["{\"@type\":\"/opinit.opchild.v1.MsgFinalizeTokenDeposit\",\"sender\":\"\",\"from\":\"\",\"to\":\"\",\"amount\":{\"denom\":\"\",\"amount\":\"0\"},\"sequence\":\"0\",\"height\":\"0\",\"base_denom\":\"\",\"data\":null}"],"timestamp":1,"save":true}`),
				},
			},
			err: false,
		},
		{
			name:          "empty changes",
			child:         NewMockChild(nil, childCodec, "sender0", "sender1", 1),
			msgQueue:      nil,
			processedMsgs: nil,
			dbChanges:     nil,
			endBlockArgs: nodetypes.EndBlockArgs{
				Block: cmtproto.Block{
					Header: cmtproto.Header{
						Height: 15,
					},
				},
			},
			expectedProcessedMsgs: nil,
			expectedDB: []types.KV{
				{
					Key:   []byte("test_host/synced_height"),
					Value: []byte("15"),
				},
			},
			err: false,
		},
		{
			name:  "child no broadcaster",
			child: NewMockChild(nil, childCodec, "", "", 1),
			msgQueue: map[string][]sdk.Msg{
				"sender0": {&opchildtypes.MsgFinalizeTokenDeposit{}},
			},
			processedMsgs: []btypes.ProcessedMsgs{
				{
					Sender:    "sender1",
					Msgs:      []sdk.Msg{&opchildtypes.MsgUpdateOracle{}},
					Timestamp: 10000,
					Save:      true,
				},
			},
			dbChanges: nil,
			endBlockArgs: nodetypes.EndBlockArgs{
				Block: cmtproto.Block{
					Header: cmtproto.Header{
						Height: 10,
					},
				},
			},
			expectedProcessedMsgs: nil,
			expectedDB: []types.KV{
				{
					Key:   []byte("test_host/synced_height"),
					Value: []byte("10"),
				},
			},
			err: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := db.NewMemDB()
			defer func() {
				require.NoError(t, db.Close())
			}()
			hostdb := db.WithPrefix([]byte("test_host"))
			require.NoError(t, err)
			hostNode := node.NewTestNode(nodetypes.NodeConfig{}, hostdb, nil, nil, nil, nil)
			tc.child.db = db.WithPrefix([]byte("test_child"))

			h := Host{
				BaseHost: hostprovider.NewTestBaseHost(0, hostNode, ophosttypes.QueryBridgeResponse{}, nodetypes.NodeConfig{}, nil),
				child:    tc.child,
				stage:    hostdb.NewStage(),
			}

			for sender, msgs := range tc.msgQueue {
				for _, msg := range msgs {
					h.AppendMsgQueue(msg, sender)
				}
			}
			for _, processedMsgs := range tc.processedMsgs {
				h.AppendProcessedMsgs(processedMsgs)
			}

			for _, kv := range tc.dbChanges {
				err := h.stage.Set(kv.Key, kv.Value)
				require.NoError(t, err)
			}

			err = h.endBlockHandler(types.Context{}, tc.endBlockArgs)
			if !tc.err {
				require.NoError(t, err)
				for i := range tc.expectedProcessedMsgs {
					expectedMsg, err := tc.expectedProcessedMsgs[i].MarshalInterfaceJSON(childCodec)
					require.NoError(t, err)
					actualMsg, err := tc.child.processedMsgs[i].MarshalInterfaceJSON(childCodec)
					require.NoError(t, err)
					require.Equal(t, expectedMsg, actualMsg)
				}
				for _, kv := range tc.expectedDB {
					value, err := db.Get(kv.Key)
					require.NoError(t, err)
					require.Equal(t, kv.Value, value)
				}
			} else {
				require.Error(t, err)
			}
		})
	}
	require.NoError(t, err)
}

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
