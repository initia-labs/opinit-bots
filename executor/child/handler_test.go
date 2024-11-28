package child

import (
	"testing"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/merkle"
	merkletypes "github.com/initia-labs/opinit-bots/merkle/types"
	"github.com/initia-labs/opinit-bots/node"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestBeginBlockHandler(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)
	childDB := db.WithPrefix([]byte("test_child"))
	childNode := node.NewTestNode(nodetypes.NodeConfig{}, childDB, nil, nil, nil, nil)

	err = childDB.Set(
		append([]byte("working_tree/"), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05}...),
		[]byte(`{"version":5,"index":1,"leaf_count":2,"start_leaf_index":1,"last_siblings":{},"done":false}`),
	)
	require.NoError(t, err)

	mk, err := merkle.NewMerkle(ophosttypes.GenerateNodeHash)
	require.NoError(t, err)

	ch := Child{
		BaseChild: childprovider.NewTestBaseChild(0, childNode, mk, ophosttypes.QueryBridgeResponse{
			BridgeId: 1,
		}, nil, nodetypes.NodeConfig{}),
		host:  NewMockHost(nil, nil, 1, "", nil),
		stage: db.NewStage(),
	}

	msgQueue := ch.GetMsgQueue()
	require.Empty(t, msgQueue["init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5"])
	ch.AppendMsgQueue(ophosttypes.NewMsgProposeOutput("init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", 1, 1, 1, []byte("oracle_tx")), "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5")
	msgQueue = ch.GetMsgQueue()
	require.Len(t, msgQueue["init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5"], 1)

	msgs := ch.GetProcessedMsgs()
	require.Empty(t, msgs)
	ch.AppendProcessedMsgs(btypes.ProcessedMsgs{})
	msgs = ch.GetProcessedMsgs()
	require.Len(t, msgs, 1)

	require.Equal(t, 0, ch.stage.Len())
	err = ch.stage.Set([]byte("key"), []byte("value"))
	require.NoError(t, err)
	require.Equal(t, 1, ch.stage.Len())

	err = ch.beginBlockHandler(types.Context{}, nodetypes.BeginBlockArgs{
		Block: cmtproto.Block{
			Header: cmtproto.Header{
				Height: 6,
			},
		},
	})
	require.NoError(t, err)

	msgQueue = ch.GetMsgQueue()
	require.Empty(t, msgQueue["init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5"])

	msgs = ch.GetProcessedMsgs()
	require.Empty(t, msgs)

	require.Equal(t, 0, ch.stage.Len())
}

func TestEndBlockHandler(t *testing.T) {
	hostCodec, _, err := hostprovider.GetCodec("init")

	mockCount := int64(0)
	mockTimestampFetcher := func() int64 {
		mockCount++
		return mockCount
	}
	types.CurrentNanoTimestamp = mockTimestampFetcher

	cases := []struct {
		name                  string
		host                  *mockHost
		msgQueue              map[string][]sdk.Msg
		processedMsgs         []btypes.ProcessedMsgs
		dbChanges             []types.KV
		endBlockArgs          nodetypes.EndBlockArgs
		expectedProcessedMsgs []btypes.ProcessedMsgs
		expectedDB            []types.KV
		err                   bool
	}{
		{
			name: "success",
			host: NewMockHost(nil, hostCodec, 1, "sender0", nil),
			msgQueue: map[string][]sdk.Msg{
				"sender0": {&ophosttypes.MsgProposeOutput{}},
			},
			processedMsgs: []btypes.ProcessedMsgs{
				{
					Sender:    "sender0",
					Msgs:      []sdk.Msg{&ophosttypes.MsgProposeOutput{}},
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
					Sender:    "sender0",
					Msgs:      []sdk.Msg{&ophosttypes.MsgProposeOutput{}},
					Timestamp: 10000,
					Save:      true,
				},
				{
					Sender:    "sender0",
					Msgs:      []sdk.Msg{&ophosttypes.MsgProposeOutput{}},
					Timestamp: 1,
					Save:      true,
				},
			},
			expectedDB: []types.KV{
				{
					Key:   []byte("test_child/key1"),
					Value: []byte("value1"),
				},
				{
					Key:   []byte("test_child/synced_height"),
					Value: []byte("10"),
				},
				{
					Key:   append([]byte("test_host/processed_msgs/"), []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x27, 0x10}...),
					Value: []byte(`{"sender":"sender0","msgs":["{\"@type\":\"/opinit.ophost.v1.MsgProposeOutput\",\"proposer\":\"\",\"bridge_id\":\"0\",\"output_index\":\"0\",\"l2_block_number\":\"0\",\"output_root\":null}"],"timestamp":10000,"save":true}`),
				},
				{
					Key:   append([]byte("test_host/processed_msgs/"), []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1}...),
					Value: []byte(`{"sender":"sender0","msgs":["{\"@type\":\"/opinit.ophost.v1.MsgProposeOutput\",\"proposer\":\"\",\"bridge_id\":\"0\",\"output_index\":\"0\",\"l2_block_number\":\"0\",\"output_root\":null}"],"timestamp":1,"save":true}`),
				},
			},
			err: false,
		},
		{
			name:          "empty changes",
			host:          NewMockHost(nil, hostCodec, 1, "sender0", nil),
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
					Key:   []byte("test_child/synced_height"),
					Value: []byte("15"),
				},
			},
			err: false,
		},
		{
			name: "host no broadcaster",
			host: NewMockHost(nil, hostCodec, 1, "", nil),
			msgQueue: map[string][]sdk.Msg{
				"sender0": {&ophosttypes.MsgProposeOutput{}},
			},
			processedMsgs: []btypes.ProcessedMsgs{
				{
					Sender:    "sender0",
					Msgs:      []sdk.Msg{&ophosttypes.MsgProposeOutput{}},
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
					Key:   []byte("test_child/synced_height"),
					Value: []byte("10"),
				},
			},
			err: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			basedb, err := db.NewMemDB()
			require.NoError(t, err)

			childdb := basedb.WithPrefix([]byte("test_child"))
			childNode := node.NewTestNode(nodetypes.NodeConfig{}, childdb, nil, nil, nil, nil)
			stage := childdb.NewStage().(*db.Stage)
			mk, err := merkle.NewMerkle(ophosttypes.GenerateNodeHash)
			require.NoError(t, err)
			err = mk.PrepareWorkingTree(merkletypes.TreeInfo{
				Version:        5,
				Index:          1,
				LeafCount:      2,
				StartLeafIndex: 1,
				LastSiblings:   map[uint8][]byte{},
				Done:           false,
			})
			require.NoError(t, err)
			tc.host.db = basedb.WithPrefix([]byte("test_host"))
			ch := Child{
				BaseChild: childprovider.NewTestBaseChild(0, childNode, mk, ophosttypes.QueryBridgeResponse{}, nil, nodetypes.NodeConfig{}),
				host:      tc.host,
				stage:     stage,
			}

			for sender, msgs := range tc.msgQueue {
				for _, msg := range msgs {
					ch.AppendMsgQueue(msg, sender)
				}
			}
			for _, processedMsgs := range tc.processedMsgs {
				ch.AppendProcessedMsgs(processedMsgs)
			}

			for _, kv := range tc.dbChanges {
				err := ch.stage.Set(kv.Key, kv.Value)
				require.NoError(t, err)
			}

			err = ch.endBlockHandler(types.Context{}, tc.endBlockArgs)
			if !tc.err {
				require.NoError(t, err)
				for i := range tc.expectedProcessedMsgs {
					expectedMsg, err := tc.expectedProcessedMsgs[i].MarshalInterfaceJSON(hostCodec)
					require.NoError(t, err)
					actualMsg, err := tc.host.processedMsgs[i].MarshalInterfaceJSON(hostCodec)
					require.NoError(t, err)
					require.Equal(t, expectedMsg, actualMsg)
				}
				for _, kv := range tc.expectedDB {
					value, err := basedb.Get(kv.Key)
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
