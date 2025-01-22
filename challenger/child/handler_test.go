package child

import (
	"context"
	"testing"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	eventhandler "github.com/initia-labs/opinit-bots/challenger/eventhandler"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/merkle"
	merkletypes "github.com/initia-labs/opinit-bots/merkle/types"
	"github.com/initia-labs/opinit-bots/node"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBeginBlockHandler(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)
	childDB := db.WithPrefix([]byte("test_child"))
	childNode := node.NewTestNode(nodetypes.NodeConfig{}, childDB, nil, nil, nil, nil)

	err = childDB.Set(
		append([]byte("working_tree/"), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x09}...),
		[]byte(`{"version":9,"index":1,"leaf_count":2,"start_leaf_index":1,"last_siblings":{},"done":false}`),
	)
	require.NoError(t, err)

	mk, err := merkle.NewMerkle(ophosttypes.GenerateNodeHash)
	require.NoError(t, err)

	mockHost := NewMockHost(nil, 1)
	mockHost.SetSyncedOutput(1, 1, &ophosttypes.Output{
		OutputRoot:    []byte("output_root"),
		L2BlockNumber: 15,
		L1BlockTime:   time.Unix(0, 10000).UTC(),
		L1BlockNumber: 1,
	})

	ch := Child{
		BaseChild: childprovider.NewTestBaseChild(0, childNode, mk, ophosttypes.QueryBridgeResponse{
			BridgeId: 1,
		}, nil, nodetypes.NodeConfig{}),
		host:       mockHost,
		stage:      db.NewStage(),
		eventQueue: make([]challengertypes.ChallengeEvent, 0),
	}

	require.Empty(t, ch.eventQueue)
	ch.eventQueue = append(ch.eventQueue, &challengertypes.Deposit{
		EventType: "Deposit",
	})
	require.Len(t, ch.eventQueue, 1)

	require.Equal(t, 0, ch.stage.Len())
	err = ch.stage.Set([]byte("key"), []byte("value"))
	require.NoError(t, err)
	require.Equal(t, 1, ch.stage.Len())

	err = ch.beginBlockHandler(types.Context{}, nodetypes.BeginBlockArgs{
		Block: cmtproto.Block{
			Header: cmtproto.Header{
				Height: 10,
			},
		},
	})
	require.NoError(t, err)

	require.Empty(t, ch.eventQueue)
	require.Equal(t, 0, ch.stage.Len())
}

func TestEndBlockHandler(t *testing.T) {
	mockCount := int64(0)
	mockTimestampFetcher := func() int64 {
		mockCount++
		return mockCount
	}
	types.CurrentNanoTimestamp = mockTimestampFetcher

	cases := []struct {
		name                  string
		host                  *mockHost
		challenger            *mockChallenger
		pendingEvents         []challengertypes.ChallengeEvent
		eventQueue            []challengertypes.ChallengeEvent
		dbChanges             []types.KV
		endBlockArgs          nodetypes.EndBlockArgs
		expectedPendingEvents []challengertypes.ChallengeEvent
		expectedChallenges    []challengertypes.Challenge
		expectedDB            []types.KV
		err                   bool
	}{
		{
			name:          "no pending events",
			host:          NewMockHost(nil, 1),
			challenger:    NewMockChallenger(nil),
			pendingEvents: []challengertypes.ChallengeEvent{},
			eventQueue: []challengertypes.ChallengeEvent{
				&challengertypes.Output{
					EventType:     "Output",
					L2BlockNumber: 15,
					OutputIndex:   1,
					OutputRoot:    []byte(""),
					Time:          time.Unix(0, 10000).UTC(),
					Timeout:       false,
				},
				&challengertypes.Deposit{
					EventType:     "Deposit",
					Sequence:      1,
					L1BlockHeight: 5,
					From:          "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
					To:            "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
					L1Denom:       "l1Denom",
					Amount:        "100l2denom",
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
			expectedPendingEvents: []challengertypes.ChallengeEvent{},
			expectedChallenges:    []challengertypes.Challenge{},
			expectedDB:            []types.KV{},
			err:                   true,
		},
		{
			name:       "existing deposit events",
			host:       NewMockHost(nil, 1),
			challenger: NewMockChallenger(nil),
			pendingEvents: []challengertypes.ChallengeEvent{
				&challengertypes.Deposit{
					EventType:     "Deposit",
					Sequence:      1,
					L1BlockHeight: 5,
					From:          "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
					To:            "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
					L1Denom:       "l1Denom",
					Amount:        "100l2denom",
					Time:          time.Unix(0, 9000).UTC(),
					Timeout:       false,
				},
				&challengertypes.Deposit{
					EventType:     "Deposit",
					Sequence:      2,
					L1BlockHeight: 5,
					From:          "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
					To:            "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
					L1Denom:       "l1Denom",
					Amount:        "100l2denom",
					Time:          time.Unix(0, 9000).UTC(),
					Timeout:       false,
				},
				&challengertypes.Deposit{
					EventType:     "Deposit",
					Sequence:      3,
					L1BlockHeight: 5,
					From:          "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
					To:            "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
					L1Denom:       "l1Denom",
					Amount:        "100l2denom",
					Time:          time.Unix(0, 9000).UTC(),
					Timeout:       false,
				},
			},
			eventQueue: []challengertypes.ChallengeEvent{
				&challengertypes.Deposit{
					EventType:     "Deposit",
					Sequence:      1,
					L1BlockHeight: 5,
					From:          "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
					To:            "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
					L1Denom:       "l1Denom",
					Amount:        "100l2denom",
					Time:          time.Unix(0, 10000).UTC(),
					Timeout:       false,
				},
				&challengertypes.Deposit{
					EventType:     "Deposit",
					Sequence:      2,
					L1BlockHeight: 5,
					From:          "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
					To:            "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
					L1Denom:       "l1Denom",
					Amount:        "100l2denom",
					Time:          time.Unix(0, 10000).UTC(),
					Timeout:       false,
				},
			},
			dbChanges: []types.KV{},
			endBlockArgs: nodetypes.EndBlockArgs{
				Block: cmtproto.Block{
					Header: cmtproto.Header{
						Height: 10,
						Time:   time.Unix(0, 10000).UTC(),
					},
				},
			},

			expectedPendingEvents: []challengertypes.ChallengeEvent{
				&challengertypes.Deposit{
					EventType:     "Deposit",
					Sequence:      3,
					L1BlockHeight: 5,
					From:          "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
					To:            "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
					L1Denom:       "l1Denom",
					Amount:        "100l2denom",
					Time:          time.Unix(0, 9000).UTC(),
					Timeout:       false,
				},
			},
			expectedChallenges: []challengertypes.Challenge{},
			expectedDB: []types.KV{
				{
					Key:   []byte("test_child/synced_height"),
					Value: []byte("10"),
				},
			},
			err: false,
		},
		{
			name:       "deposit timeout",
			host:       NewMockHost(nil, 1),
			challenger: NewMockChallenger(nil),
			pendingEvents: []challengertypes.ChallengeEvent{
				&challengertypes.Deposit{
					EventType:     "Deposit",
					Sequence:      1,
					L1BlockHeight: 5,
					From:          "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
					To:            "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
					L1Denom:       "l1Denom",
					Amount:        "100l2denom",
					Time:          time.Unix(0, 9000).UTC(),
					Timeout:       false,
				},
				&challengertypes.Deposit{
					EventType:     "Deposit",
					Sequence:      2,
					L1BlockHeight: 50,
					From:          "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
					To:            "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
					L1Denom:       "l1Denom",
					Amount:        "100l2denom",
					Time:          time.Unix(0, 9000).UTC(),
					Timeout:       false,
				},
				&challengertypes.Deposit{
					EventType:     "Deposit",
					Sequence:      3,
					L1BlockHeight: 50,
					From:          "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
					To:            "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
					L1Denom:       "l1Denom",
					Amount:        "100l2denom",
					Time:          time.Unix(0, 9000).UTC(),
					Timeout:       false,
				},
			},
			eventQueue: []challengertypes.ChallengeEvent{
				&challengertypes.Deposit{
					EventType:     "Deposit",
					Sequence:      1,
					L1BlockHeight: 5,
					From:          "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
					To:            "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
					L1Denom:       "l1Denom",
					Amount:        "100l2denom",
					Time:          time.Unix(0, 20000).UTC(),
					Timeout:       false,
				},
			},
			dbChanges: []types.KV{},
			endBlockArgs: nodetypes.EndBlockArgs{
				Block: cmtproto.Block{
					Header: cmtproto.Header{
						Height: 10,
						Time:   time.Unix(0, 20000).UTC(),
					},
				},
			},

			expectedPendingEvents: []challengertypes.ChallengeEvent{
				&challengertypes.Deposit{
					EventType:     "Deposit",
					Sequence:      2,
					L1BlockHeight: 50,
					From:          "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
					To:            "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
					L1Denom:       "l1Denom",
					Amount:        "100l2denom",
					Time:          time.Unix(0, 9000).UTC(),
					Timeout:       true,
				},
				&challengertypes.Deposit{
					EventType:     "Deposit",
					Sequence:      3,
					L1BlockHeight: 50,
					From:          "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
					To:            "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
					L1Denom:       "l1Denom",
					Amount:        "100l2denom",
					Time:          time.Unix(0, 9000).UTC(),
					Timeout:       true,
				},
			},
			expectedChallenges: []challengertypes.Challenge{
				{
					EventType: "Deposit",
					Id: challengertypes.ChallengeId{
						Type: challengertypes.EventTypeDeposit,
						Id:   2,
					},
					Time: time.Unix(0, 20000).UTC(),
				},
				{
					EventType: "Deposit",
					Id: challengertypes.ChallengeId{
						Type: challengertypes.EventTypeDeposit,
						Id:   3,
					},
					Time: time.Unix(0, 20000).UTC(),
				},
			},
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
			db, err := db.NewMemDB()
			defer func() {
				require.NoError(t, db.Close())
			}()
			childdb := db.WithPrefix([]byte("test_child"))
			require.NoError(t, err)
			tc.host.db = db.WithPrefix([]byte("test_host"))
			tc.challenger.db = db

			childNode := node.NewTestNode(nodetypes.NodeConfig{}, childdb, nil, nil, nil, nil)
			mk, err := merkle.NewMerkle(ophosttypes.GenerateNodeHash)
			require.NoError(t, err)
			err = mk.PrepareWorkingTree(merkletypes.TreeInfo{
				Version:        11,
				Index:          1,
				LeafCount:      1,
				StartLeafIndex: 1,
				LastSiblings:   map[uint8][]byte{},
				Done:           false,
			})
			require.NoError(t, err)
			ch := Child{
				BaseChild:  childprovider.NewTestBaseChild(0, childNode, mk, ophosttypes.QueryBridgeResponse{}, nil, nodetypes.NodeConfig{}),
				host:       tc.host,
				challenger: tc.challenger,
				stage:      childdb.NewStage(),
				eventQueue: tc.eventQueue,

				eventHandler: eventhandler.NewChallengeEventHandler(childdb),
			}

			tc.host.SetSyncedOutput(1, 1, &ophosttypes.Output{
				OutputRoot:    []byte("output_root"),
				L2BlockNumber: 15,
				L1BlockTime:   time.Unix(0, 10000).UTC(),
				L1BlockNumber: 1,
			})

			err = ch.eventHandler.Initialize(10000)
			require.NoError(t, err)
			ch.eventHandler.SetPendingEvents(tc.pendingEvents)

			for _, kv := range tc.dbChanges {
				err := ch.stage.Set(kv.Key, kv.Value)
				require.NoError(t, err)
			}

			err = ch.endBlockHandler(types.NewContext(context.Background(), zap.NewNop(), ""), tc.endBlockArgs)
			if !tc.err {
				require.NoError(t, err)
				require.Equal(t, len(tc.expectedChallenges), len(tc.challenger.pendingChallenges))
				for i, challenge := range tc.expectedChallenges {
					require.Equal(t, challenge.EventType, tc.challenger.pendingChallenges[i].EventType)
					require.Equal(t, challenge.Id, tc.challenger.pendingChallenges[i].Id)
					require.Equal(t, challenge.Time, tc.challenger.pendingChallenges[i].Time)
				}
				require.ElementsMatch(t, tc.expectedPendingEvents, ch.eventHandler.GetAllPendingEvents())

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
