package host

import (
	"testing"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots/challenger/eventhandler"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/node"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
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

	require.Empty(t, h.eventQueue)
	h.eventQueue = append(h.eventQueue, &challengertypes.Deposit{
		EventType: "Deposit",
	})
	require.Len(t, h.eventQueue, 1)

	require.Empty(t, h.outputPendingEventQueue)
	h.outputPendingEventQueue = append(h.outputPendingEventQueue, &challengertypes.Output{
		EventType: "Output",
	})
	require.Len(t, h.outputPendingEventQueue, 1)

	require.Equal(t, 0, h.stage.Len())
	err = h.stage.Set([]byte("key"), []byte("value"))
	require.NoError(t, err)
	require.Equal(t, 1, h.stage.Len())

	err = h.beginBlockHandler(types.Context{}, nodetypes.BeginBlockArgs{})
	require.NoError(t, err)

	require.Empty(t, h.eventQueue)
	require.Empty(t, h.outputPendingEventQueue)

	require.Equal(t, 0, h.stage.Len())
}

func TestEndBlockHandler(t *testing.T) {
	cases := []struct {
		name                    string
		child                   *mockChild
		challenger              *mockChallenger
		pendingEvents           []challengertypes.ChallengeEvent
		eventQueue              []challengertypes.ChallengeEvent
		outputPendingEventQueue []challengertypes.ChallengeEvent
		dbChanges               []types.KV
		endBlockArgs            nodetypes.EndBlockArgs
		expectedPendingEvents   []challengertypes.ChallengeEvent
		expectedEventQueue      []challengertypes.ChallengeEvent
		expectedChallenges      []challengertypes.Challenge
		expectedDB              []types.KV
		err                     bool
	}{
		{
			name:          "zero pending events",
			child:         NewMockChild(nil, 1, 1),
			challenger:    NewMockChallenger(nil),
			pendingEvents: []challengertypes.ChallengeEvent{},
			eventQueue: []challengertypes.ChallengeEvent{
				&challengertypes.Deposit{
					EventType:     "Deposit",
					Sequence:      1,
					L1BlockHeight: 10,
					From:          "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
					To:            "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
					L1Denom:       "l1Denom",
					Amount:        "100l2denom",
					Time:          time.Unix(0, 10000).UTC(),
					Timeout:       false,
				},
			},
			outputPendingEventQueue: []challengertypes.ChallengeEvent{
				&challengertypes.Output{
					EventType:     "Output",
					L2BlockNumber: 2,
					OutputIndex:   1,
					OutputRoot:    []byte(""),
					Time:          time.Unix(0, 10000).UTC(),
					Timeout:       false,
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
						Time:   time.Unix(0, 10000).UTC(),
					},
				},
			},
			expectedPendingEvents: []challengertypes.ChallengeEvent{
				&challengertypes.Output{
					EventType:     "Output",
					L2BlockNumber: 2,
					OutputIndex:   1,
					OutputRoot:    []byte(""),
					Time:          time.Unix(0, 10000).UTC(),
					Timeout:       false,
				},
			},
			expectedEventQueue: []challengertypes.ChallengeEvent{
				&challengertypes.Deposit{
					EventType:     "Deposit",
					Sequence:      1,
					L1BlockHeight: 10,
					From:          "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
					To:            "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
					L1Denom:       "l1Denom",
					Amount:        "100l2denom",
					Time:          time.Unix(0, 10000).UTC(),
					Timeout:       false,
				},
			},
			expectedChallenges: []challengertypes.Challenge{},
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
					Key:   append([]byte("test_host/pending_event/"), []byte{0x1, '/', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1}...),
					Value: []byte(`{"event_type":"Output","l2_block_number":2,"output_index":1,"output_root":"","time":"1970-01-01T00:00:00.00001Z","timeout":false}`),
				},
				{
					Key:   append([]byte("test_child/pending_event/"), []byte{0x0, '/', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1}...),
					Value: []byte(`{"event_type":"Deposit","sequence":1,"l1_block_height":10,"from":"init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5","to":"init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0","l1_denom":"l1Denom","amount":"100l2denom","time":"1970-01-01T00:00:00.00001Z","timeout":false}`),
				},
			},
			err: false,
		},
		{
			name:       "update output event",
			child:      NewMockChild(nil, 1, 1),
			challenger: NewMockChallenger(nil),
			pendingEvents: []challengertypes.ChallengeEvent{
				&challengertypes.Output{
					EventType:     "Output",
					L2BlockNumber: 2,
					OutputIndex:   1,
					OutputRoot:    []byte(""),
					Time:          time.Unix(0, 10000).UTC(),
					Timeout:       false,
				},
			},
			eventQueue: []challengertypes.ChallengeEvent{},
			outputPendingEventQueue: []challengertypes.ChallengeEvent{
				&challengertypes.Output{
					EventType:     "Output",
					L2BlockNumber: 2,
					OutputIndex:   2,
					OutputRoot:    []byte(""),
					Time:          time.Unix(0, 11000).UTC(),
					Timeout:       false,
				},
			},
			dbChanges: []types.KV{},
			endBlockArgs: nodetypes.EndBlockArgs{
				Block: cmtproto.Block{
					Header: cmtproto.Header{
						Height: 10,
						Time:   time.Unix(0, 11000).UTC(),
					},
				},
			},
			expectedPendingEvents: []challengertypes.ChallengeEvent{
				&challengertypes.Output{
					EventType:     "Output",
					L2BlockNumber: 2,
					OutputIndex:   2,
					OutputRoot:    []byte(""),
					Time:          time.Unix(0, 11000).UTC(),
					Timeout:       false,
				},
			},
			expectedEventQueue: []challengertypes.ChallengeEvent{},
			expectedChallenges: []challengertypes.Challenge{},
			expectedDB: []types.KV{
				{
					Key:   []byte("test_host/synced_height"),
					Value: []byte("10"),
				},
				{
					Key:   append([]byte("test_host/pending_event/"), []byte{0x1, '/', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2}...),
					Value: []byte(`{"event_type":"Output","l2_block_number":2,"output_index":2,"output_root":"","time":"1970-01-01T00:00:00.000011Z","timeout":false}`),
				},
			},
			err: false,
		},
		{
			name:       "output event timeout",
			child:      NewMockChild(nil, 1, 1),
			challenger: NewMockChallenger(nil),
			pendingEvents: []challengertypes.ChallengeEvent{
				&challengertypes.Output{
					EventType:     "Output",
					L2BlockNumber: 1,
					OutputIndex:   1,
					OutputRoot:    []byte(""),
					Time:          time.Unix(0, 10000).UTC(),
					Timeout:       false,
				},
			},
			eventQueue:              []challengertypes.ChallengeEvent{},
			outputPendingEventQueue: []challengertypes.ChallengeEvent{},
			dbChanges:               []types.KV{},
			endBlockArgs: nodetypes.EndBlockArgs{
				Block: cmtproto.Block{
					Header: cmtproto.Header{
						Height: 10,
						Time:   time.Unix(0, 21000).UTC(),
					},
				},
			},
			expectedPendingEvents: []challengertypes.ChallengeEvent{},
			expectedEventQueue:    []challengertypes.ChallengeEvent{},
			expectedChallenges: []challengertypes.Challenge{
				{
					EventType: "Output",
					Id: challengertypes.ChallengeId{
						Type: challengertypes.EventTypeOutput,
						Id:   1,
					},
					Time: time.Unix(0, 21000).UTC(),
				},
			},
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
			tc.challenger.db = db

			h := Host{
				BaseHost:                hostprovider.NewTestBaseHost(0, hostNode, ophosttypes.QueryBridgeResponse{}, nodetypes.NodeConfig{}, nil),
				child:                   tc.child,
				challenger:              tc.challenger,
				stage:                   hostdb.NewStage(),
				eventHandler:            eventhandler.NewChallengeEventHandler(hostdb),
				eventQueue:              tc.eventQueue,
				outputPendingEventQueue: tc.outputPendingEventQueue,
			}
			err = h.eventHandler.Initialize(10000)
			require.NoError(t, err)
			h.eventHandler.SetPendingEvents(tc.pendingEvents)

			for _, kv := range tc.dbChanges {
				err := h.stage.Set(kv.Key, kv.Value)
				require.NoError(t, err)
			}

			err = h.endBlockHandler(types.Context{}, tc.endBlockArgs)
			if !tc.err {
				require.NoError(t, err)
				require.Equal(t, tc.expectedEventQueue, tc.child.pendingEvents)
				require.Equal(t, len(tc.expectedChallenges), len(tc.challenger.pendingChallenges))
				for i, challenge := range tc.expectedChallenges {
					require.Equal(t, challenge.EventType, tc.challenger.pendingChallenges[i].EventType)
					require.Equal(t, challenge.Id, tc.challenger.pendingChallenges[i].Id)
					require.Equal(t, challenge.Time, tc.challenger.pendingChallenges[i].Time)
				}
				require.ElementsMatch(t, tc.expectedPendingEvents, h.eventHandler.GetAllPendingEvents())

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
}
