package child

import (
	"context"
	"strconv"
	"testing"
	"time"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/merkle"
	merkletypes "github.com/initia-labs/opinit-bots/merkle/types"
	"github.com/initia-labs/opinit-bots/node"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func InitiateWithdrawalEvents(
	from string,
	to string,
	denom string,
	baseDenom string,
	amount sdk.Coin,
	l2Sequence uint64,
) []abcitypes.EventAttribute {
	return []abcitypes.EventAttribute{
		{
			Key:   opchildtypes.AttributeKeyFrom,
			Value: from,
		},
		{
			Key:   opchildtypes.AttributeKeyTo,
			Value: to,
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
			Key:   opchildtypes.AttributeKeyL2Sequence,
			Value: strconv.FormatUint(l2Sequence, 10),
		},
	}
}

func TestInitiateWithdrawalHandler(t *testing.T) {
	bridgeInfo := ophosttypes.QueryBridgeResponse{
		BridgeId: 1,
	}

	fullAttributes := InitiateWithdrawalEvents("from", "to", "denom", "uinit", sdk.NewInt64Coin("uinit", 10000), 1)

	cases := []struct {
		name             string
		lastWorkingTree  merkletypes.TreeInfo
		eventHandlerArgs nodetypes.EventHandlerArgs
		expectedStage    []types.KV
		expectedLog      func() (msg string, fields []zapcore.Field)
		err              bool
		panic            bool
	}{
		{
			name: "success",
			lastWorkingTree: merkletypes.TreeInfo{
				Version:        10,
				Index:          5,
				LeafCount:      0,
				StartLeafIndex: 1,
				LastSiblings:   make(map[uint8][]byte),
				Done:           false,
			},
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: InitiateWithdrawalEvents("from", "to", "denom", "uinit", sdk.NewInt64Coin("uinit", 10000), 1),
			},
			expectedStage: []types.KV{
				{
					Key:   append([]byte("/test_child/withdrawal_sequence/"), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}...),
					Value: []byte(`{"sequence":1,"from":"from","to":"to","amount":10000,"base_denom":"uinit","withdrawal_hash":"V+7ukqwrq0Ba6kj63TEZ1C7m4Ze7pqERmid/OQtNneY="}`),
				},
				{
					Key:   append([]byte("/test_child/withdrawal_address/to/"), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}...),
					Value: []byte(`1`),
				},
				{ // local node 0
					Key:   append([]byte("/test_child/node/"), []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}...),
					Value: []byte{0x57, 0xee, 0xee, 0x92, 0xac, 0x2b, 0xab, 0x40, 0x5a, 0xea, 0x48, 0xfa, 0xdd, 0x31, 0x19, 0xd4, 0x2e, 0xe6, 0xe1, 0x97, 0xbb, 0xa6, 0xa1, 0x11, 0x9a, 0x27, 0x7f, 0x39, 0x0b, 0x4d, 0x9d, 0xe6},
				},
			},
			expectedLog: func() (msg string, fields []zapcore.Field) {
				msg = "initiate token withdrawal"
				fields = []zapcore.Field{
					zap.Uint64("l2_sequence", 1),
					zap.String("from", "from"),
					zap.String("to", "to"),
					zap.Uint64("amount", 10000),
					zap.String("base_denom", "uinit"),
					zap.String("withdrawal", "V+7ukqwrq0Ba6kj63TEZ1C7m4Ze7pqERmid/OQtNneY="),
				}
				return msg, fields
			},
			err:   false,
			panic: false,
		},
		{
			name: "second withdrawal",
			lastWorkingTree: merkletypes.TreeInfo{
				Version:        10,
				Index:          5,
				LeafCount:      1,
				StartLeafIndex: 100,
				LastSiblings: map[uint8][]byte{
					0: {0x5e, 0xc5, 0xb8, 0x13, 0x43, 0xb9, 0x76, 0xbb, 0xef, 0x23, 0xbc, 0x6e, 0x6a, 0xbe, 0x44, 0xa6, 0xa7, 0x17, 0x8c, 0x66, 0xae, 0xfd, 0x78, 0xe8, 0xd8, 0x1c, 0x73, 0x36, 0xf3, 0x32, 0xb6, 0x31},
				},
				Done: false,
			},
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: InitiateWithdrawalEvents("from", "to", "denom", "uinit", sdk.NewInt64Coin("uinit", 10000), 101),
			},
			expectedStage: []types.KV{
				{
					Key:   append([]byte("/test_child/withdrawal_sequence/"), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x65}...),
					Value: []byte(`{"sequence":101,"from":"from","to":"to","amount":10000,"base_denom":"uinit","withdrawal_hash":"Hzn58U22rfXK2VZCOIFzjudpdYkw5v0eZ2QnspIFlBs="}`),
				},
				{
					Key:   append([]byte("/test_child/withdrawal_address/to/"), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x65}...),
					Value: []byte(`101`),
				},
				{ // local node 1
					Key:   append([]byte("/test_child/node/"), []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1}...),
					Value: []byte{0x1f, 0x39, 0xf9, 0xf1, 0x4d, 0xb6, 0xad, 0xf5, 0xca, 0xd9, 0x56, 0x42, 0x38, 0x81, 0x73, 0x8e, 0xe7, 0x69, 0x75, 0x89, 0x30, 0xe6, 0xfd, 0x1e, 0x67, 0x64, 0x27, 0xb2, 0x92, 0x05, 0x94, 0x1b},
				},
				{ // height 1, local node 0
					Key:   append([]byte("/test_child/node/"), []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}...),
					Value: []byte{0x06, 0x90, 0x8d, 0x0d, 0x10, 0x0f, 0x55, 0x78, 0xaa, 0x12, 0x81, 0xa1, 0x72, 0xbf, 0x46, 0x65, 0x09, 0xd3, 0xa0, 0x3c, 0xb2, 0x4c, 0xa1, 0xb4, 0x32, 0xb9, 0x11, 0x71, 0x5e, 0x10, 0xa9, 0xb6},
				},
			},
			expectedLog: func() (msg string, fields []zapcore.Field) {
				msg = "initiate token withdrawal"
				fields = []zapcore.Field{
					zap.Uint64("l2_sequence", 101),
					zap.String("from", "from"),
					zap.String("to", "to"),
					zap.Uint64("amount", 10000),
					zap.String("base_denom", "uinit"),
					zap.String("withdrawal", "Hzn58U22rfXK2VZCOIFzjudpdYkw5v0eZ2QnspIFlBs="),
				}
				return msg, fields
			},
			err:   false,
			panic: false,
		},
		{
			name: "panic: working tree leaf count mismatch",
			lastWorkingTree: merkletypes.TreeInfo{
				Version:        9,
				Index:          5,
				LeafCount:      0,
				StartLeafIndex: 100,
				LastSiblings:   make(map[uint8][]byte),
				Done:           false,
			},
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: InitiateWithdrawalEvents("from", "to", "denom", "uinit", sdk.NewInt64Coin("uinit", 10000), 101),
			},
			expectedStage: nil,
			expectedLog:   nil,
			err:           false,
			panic:         true,
		},
		{
			name: "missing event attribute from",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: fullAttributes[1:],
			},
			err: true,
		},
		{
			name: "missing event attribute to",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: append(fullAttributes[:1], fullAttributes[2:]...),
			},
			err: true,
		},
		{
			name: "missing event attribute denom",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: append(fullAttributes[:2], fullAttributes[3:]...),
			},
			err: true,
		},
		{
			name: "missing event attribute base denom",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: append(fullAttributes[:3], fullAttributes[4:]...),
			},
			err: true,
		},
		{
			name: "missing event attribute amount",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: append(fullAttributes[:4], fullAttributes[5:]...),
			},
			err: true,
		},
		{
			name: "missing event attribute l2 sequence",
			eventHandlerArgs: nodetypes.EventHandlerArgs{
				EventAttributes: fullAttributes[:5],
			},
			err: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logger, observedLogs := logCapturer()
			ctx := types.NewContext(context.Background(), logger, "")

			basedb, err := db.NewMemDB()
			require.NoError(t, err)

			childdb := basedb.WithPrefix([]byte("test_child"))

			childNode := node.NewTestNode(nodetypes.NodeConfig{}, childdb, nil, nil, nil, nil)

			mk, err := merkle.NewMerkle(ophosttypes.GenerateNodeHash)
			require.NoError(t, err)
			err = mk.PrepareWorkingTree(tc.lastWorkingTree)
			require.NoError(t, err)

			stage := childdb.NewStage().(*db.Stage)
			ch := Child{
				BaseChild: childprovider.NewTestBaseChild(0, childNode, mk, bridgeInfo, nil, nodetypes.NodeConfig{}),
				stage:     stage,
			}

			if tc.panic {
				require.Panics(t, func() {
					ch.initiateWithdrawalHandler(ctx, tc.eventHandlerArgs) //nolint
				})
				return
			}

			err = ch.initiateWithdrawalHandler(ctx, tc.eventHandlerArgs)
			if !tc.err {
				require.NoError(t, err)
				logs := observedLogs.TakeAll()
				if tc.expectedLog != nil {
					require.Len(t, logs, 1)

					expectedMsg, expectedFields := tc.expectedLog()
					require.Equal(t, expectedMsg, logs[0].Message)
					require.Equal(t, expectedFields, logs[0].Context)
				} else {
					require.Len(t, logs, 0)
				}

				if tc.expectedStage != nil {
					allkvs := stage.All()
					for _, kv := range tc.expectedStage {
						require.Equal(t, kv.Value, allkvs[string(kv.Key)])
					}
				}
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestPrepareTree(t *testing.T) {
	bridgeInfo := ophosttypes.QueryBridgeResponse{
		BridgeId: 1,
	}

	cases := []struct {
		name                  string
		childDBState          []types.KV
		blockHeight           int64
		initializeTreeFnMaker func(*merkle.Merkle) func(int64) (bool, error)
		expected              merkletypes.TreeInfo
		err                   bool
		panic                 bool
	}{
		{
			name: "new height 6",
			childDBState: []types.KV{
				{
					Key:   append([]byte("working_tree/"), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05}...),
					Value: []byte(`{"version":5,"index":2,"leaf_count":0,"start_leaf_index":1,"last_siblings":{},"done":false}`),
				},
			},
			blockHeight:           6,
			initializeTreeFnMaker: nil,
			expected: merkletypes.TreeInfo{
				Version:        6,
				Index:          2,
				LeafCount:      0,
				StartLeafIndex: 1,
				LastSiblings:   make(map[uint8][]byte),
				Done:           false,
			},
			err:   false,
			panic: false,
		},
		{
			name:                  "no tree height 5, new height 6, no initializeTreeFn",
			childDBState:          nil,
			blockHeight:           6,
			initializeTreeFnMaker: nil,
			expected: merkletypes.TreeInfo{
				Version:        6,
				Index:          2,
				LeafCount:      0,
				StartLeafIndex: 1,
				LastSiblings:   make(map[uint8][]byte),
				Done:           false,
			},
			err:   false,
			panic: true,
		},
		{
			name:         "no tree height 5, new height 6, no initializing tree",
			childDBState: nil,
			blockHeight:  6,
			initializeTreeFnMaker: func(m *merkle.Merkle) func(i int64) (bool, error) {
				return func(i int64) (bool, error) {
					return false, nil
				}
			},
			expected: merkletypes.TreeInfo{
				Version:        6,
				Index:          2,
				LeafCount:      0,
				StartLeafIndex: 1,
				LastSiblings:   make(map[uint8][]byte),
				Done:           false,
			},
			err:   false,
			panic: true,
		},
		{
			name: "tree done at 5, new height 6",
			childDBState: []types.KV{
				{
					Key:   append([]byte("working_tree/"), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05}...),
					Value: []byte(`{"version":5,"index":2,"leaf_count":2,"start_leaf_index":1,"last_siblings":{},"done":true}`),
				},
			},
			blockHeight:           6,
			initializeTreeFnMaker: nil,
			expected: merkletypes.TreeInfo{
				Version:        6,
				Index:          3,
				LeafCount:      0,
				StartLeafIndex: 3,
				LastSiblings:   make(map[uint8][]byte),
				Done:           false,
			},
			err:   false,
			panic: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			basedb, err := db.NewMemDB()
			require.NoError(t, err)

			childdb := basedb.WithPrefix([]byte("test_child"))

			childNode := node.NewTestNode(nodetypes.NodeConfig{}, childdb, nil, nil, nil, nil)

			mk, err := merkle.NewMerkle(ophosttypes.GenerateNodeHash)
			require.NoError(t, err)

			var initializeFn func(i int64) (bool, error)
			if tc.initializeTreeFnMaker != nil {
				initializeFn = tc.initializeTreeFnMaker(mk)
			}

			ch := Child{
				BaseChild: childprovider.NewTestBaseChild(0, childNode, mk, bridgeInfo, initializeFn, nodetypes.NodeConfig{}),
			}

			for _, kv := range tc.childDBState {
				err = childdb.Set(kv.Key, kv.Value)
				require.NoError(t, err)
			}

			if tc.panic {
				require.Panics(t, func() {
					ch.prepareTree(tc.blockHeight) //nolint
				})
				return
			}
			err = ch.prepareTree(tc.blockHeight)
			if !tc.err {
				require.NoError(t, err)

				tree, err := mk.WorkingTree()
				require.NoError(t, err)

				require.Equal(t, tc.expected, tree)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestPrepareOutput(t *testing.T) {
	cases := []struct {
		name            string
		bridgeInfo      ophosttypes.QueryBridgeResponse
		hostOutputs     map[uint64]ophosttypes.Output
		lastWorkingTree merkletypes.TreeInfo
		expected        func() (lastOutputTime time.Time, nextOutputTime time.Time, finalizingBlockHeight int64)
		err             bool
	}{
		{
			name: "no output, index 1",
			bridgeInfo: ophosttypes.QueryBridgeResponse{
				BridgeId: 1,
				BridgeConfig: ophosttypes.BridgeConfig{
					SubmissionInterval: 100,
				},
			},
			hostOutputs: map[uint64]ophosttypes.Output{},
			lastWorkingTree: merkletypes.TreeInfo{
				Version:        9,
				Index:          1,
				LeafCount:      2,
				StartLeafIndex: 1,
				LastSiblings:   make(map[uint8][]byte),
				Done:           false,
			},
			expected: func() (lastOutputTime time.Time, nextOutputTime time.Time, finalizingBlockHeight int64) {
				return time.Time{}, time.Time{}, 0
			},
			err: false,
		},
		{
			name: "no output, index 3", // chain rolled back
			bridgeInfo: ophosttypes.QueryBridgeResponse{
				BridgeId: 1,
				BridgeConfig: ophosttypes.BridgeConfig{
					SubmissionInterval: 100,
				},
			},
			hostOutputs: map[uint64]ophosttypes.Output{},
			lastWorkingTree: merkletypes.TreeInfo{
				Version:        9,
				Index:          3,
				LeafCount:      2,
				StartLeafIndex: 1,
				LastSiblings:   make(map[uint8][]byte),
				Done:           false,
			},
			expected: func() (lastOutputTime time.Time, nextOutputTime time.Time, finalizingBlockHeight int64) {
				return time.Time{}, time.Time{}, 0
			},
			err: true,
		},
		{
			name: "outputs {1}, index 1", // sync
			bridgeInfo: ophosttypes.QueryBridgeResponse{
				BridgeId: 1,
				BridgeConfig: ophosttypes.BridgeConfig{
					SubmissionInterval: 100,
				},
			},
			hostOutputs: map[uint64]ophosttypes.Output{
				1: {
					L1BlockTime:   time.Time{},
					L2BlockNumber: 10,
				},
			},
			lastWorkingTree: merkletypes.TreeInfo{
				Version:        9,
				Index:          1,
				LeafCount:      2,
				StartLeafIndex: 1,
				LastSiblings:   make(map[uint8][]byte),
				Done:           false,
			},
			expected: func() (lastOutputTime time.Time, nextOutputTime time.Time, finalizingBlockHeight int64) {
				return time.Time{}, time.Time{}, 10
			},
			err: false,
		},
		{
			name: "outputs {1}, index 2",
			bridgeInfo: ophosttypes.QueryBridgeResponse{
				BridgeId: 1,
				BridgeConfig: ophosttypes.BridgeConfig{
					SubmissionInterval: 300,
				},
			},
			hostOutputs: map[uint64]ophosttypes.Output{
				1: {
					L1BlockTime:   time.Unix(0, 10000),
					L2BlockNumber: 10,
				},
			},
			lastWorkingTree: merkletypes.TreeInfo{
				Version:        9,
				Index:          2,
				LeafCount:      2,
				StartLeafIndex: 1,
				LastSiblings:   make(map[uint8][]byte),
				Done:           false,
			},
			expected: func() (lastOutputTime time.Time, nextOutputTime time.Time, finalizingBlockHeight int64) {
				return time.Unix(0, 10000), time.Unix(0, 10200), 0
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

			mk, err := merkle.NewMerkle(ophosttypes.GenerateNodeHash)
			require.NoError(t, err)
			err = mk.PrepareWorkingTree(tc.lastWorkingTree)
			require.NoError(t, err)

			mockHost := NewMockHost(nil, nil, tc.bridgeInfo.BridgeId, "", tc.hostOutputs)

			ch := Child{
				BaseChild: childprovider.NewTestBaseChild(0, childNode, mk, tc.bridgeInfo, nil, nodetypes.NodeConfig{}),
				host:      mockHost,
			}

			err = ch.prepareOutput(context.TODO())
			if !tc.err {
				require.NoError(t, err)

				expectedLastOutputTime, expectedNextOutputTime, expectedFinalizingBlockHeight := tc.expected()
				require.Equal(t, expectedLastOutputTime, ch.lastOutputTime)
				require.Equal(t, expectedNextOutputTime, ch.nextOutputTime)
				require.Equal(t, expectedFinalizingBlockHeight, ch.finalizingBlockHeight)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestHandleTree(t *testing.T) {
	bridgeInfo := ophosttypes.QueryBridgeResponse{
		BridgeId: 1,
		BridgeConfig: ophosttypes.BridgeConfig{
			SubmissionInterval: 300,
		},
	}
	blockId := []byte("test_block_id")

	cases := []struct {
		name                  string
		blockHeight           int64
		latestHeight          int64
		blockHeader           cmtproto.Header
		lastWorkingTree       merkletypes.TreeInfo
		lastOutputTime        time.Time
		nextOutputTime        time.Time
		finalizingBlockHeight int64

		expected      func() (storageRoot []byte, lastOutputTime time.Time, nextOutputTime time.Time, finalizingBlockHeight int64)
		expectedStage []types.KV
		err           bool
		panic         bool
	}{
		{
			name:         "current height 5, latest height 5, no leaf", // not saving finalized tree
			blockHeight:  5,
			latestHeight: 5,
			blockHeader: cmtproto.Header{
				Time: time.Unix(0, 10100),
			},
			lastWorkingTree: merkletypes.TreeInfo{
				Version:        4,
				Index:          3,
				LeafCount:      0,
				StartLeafIndex: 10,
				LastSiblings:   make(map[uint8][]byte),
			},
			lastOutputTime:        time.Time{},
			nextOutputTime:        time.Unix(0, 10000),
			finalizingBlockHeight: 0,

			expected: func() (storageRoot []byte, lastOutputTime time.Time, nextOutputTime time.Time, finalizingBlockHeight int64) {
				return []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
					time.Unix(0, 10100), time.Unix(0, 10300), 0
			},
			expectedStage: []types.KV{
				{
					Key:   append([]byte("/test_child/working_tree/"), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05}...),
					Value: []byte(`{"version":5,"index":3,"leaf_count":0,"start_leaf_index":10,"last_siblings":{},"done":true}`),
				},
			},
			err:   false,
			panic: false,
		},
		{
			name:         "current height 5, latest height 5, 2 leaves",
			blockHeight:  5,
			latestHeight: 5,
			blockHeader: cmtproto.Header{
				Time: time.Unix(0, 10100),
			},
			lastWorkingTree: merkletypes.TreeInfo{
				Version:        4,
				Index:          3,
				LeafCount:      2,
				StartLeafIndex: 10,
				LastSiblings: map[uint8][]byte{
					0: {0xf7, 0x58, 0xe5, 0x5d, 0xb1, 0x30, 0x74, 0x4b, 0x05, 0xad, 0x66, 0x94, 0xb2, 0x8b, 0xe4, 0xab, 0x73, 0x0d, 0xe0, 0xdc, 0x09, 0xde, 0x5c, 0x0c, 0x42, 0xab, 0x64, 0x66, 0xc8, 0x06, 0xdc, 0x10},
					1: {0x50, 0x26, 0x55, 0x2e, 0x7b, 0x21, 0xca, 0xb5, 0x27, 0xe4, 0x16, 0x9e, 0x66, 0x46, 0x02, 0xb8, 0x5d, 0x03, 0x67, 0x0b, 0xb5, 0x57, 0xe3, 0x29, 0x18, 0xd9, 0x33, 0xe3, 0xd5, 0x92, 0x5c, 0x7e},
				},
			},
			lastOutputTime:        time.Time{},
			nextOutputTime:        time.Unix(0, 10000),
			finalizingBlockHeight: 0,

			expected: func() (storageRoot []byte, lastOutputTime time.Time, nextOutputTime time.Time, finalizingBlockHeight int64) {
				return []byte{0x50, 0x26, 0x55, 0x2e, 0x7b, 0x21, 0xca, 0xb5, 0x27, 0xe4, 0x16, 0x9e, 0x66, 0x46, 0x02, 0xb8, 0x5d, 0x03, 0x67, 0x0b, 0xb5, 0x57, 0xe3, 0x29, 0x18, 0xd9, 0x33, 0xe3, 0xd5, 0x92, 0x5c, 0x7e},
					time.Unix(0, 10100), time.Unix(0, 10300), 0
			},
			expectedStage: []types.KV{
				{
					Key:   append([]byte("/test_child/working_tree/"), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05}...),
					Value: []byte(`{"version":5,"index":3,"leaf_count":2,"start_leaf_index":10,"last_siblings":{"0":"91jlXbEwdEsFrWaUsovkq3MN4NwJ3lwMQqtkZsgG3BA=","1":"UCZVLnshyrUn5BaeZkYCuF0DZwu1V+MpGNkz49WSXH4="},"done":true}`),
				},
				{
					Key:   append([]byte("/test_child/finalized_tree/"), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0a}...),
					Value: []byte(`{"tree_index":3,"tree_height":1,"root":"UCZVLnshyrUn5BaeZkYCuF0DZwu1V+MpGNkz49WSXH4=","start_leaf_index":10,"leaf_count":2,"extra_data":"eyJibG9ja19udW1iZXIiOjUsImJsb2NrX2hhc2giOiJkR1Z6ZEY5aWJHOWphMTlwWkE9PSJ9"}`),
				},
			},
			err:   false,
			panic: false,
		},
		{
			name:         "current height 5, latest height 5, 3 leaves",
			blockHeight:  5,
			latestHeight: 5,
			blockHeader: cmtproto.Header{
				Time: time.Unix(0, 10100),
			},
			lastWorkingTree: merkletypes.TreeInfo{
				Version:        4,
				Index:          3,
				LeafCount:      3,
				StartLeafIndex: 10,
				LastSiblings: map[uint8][]byte{
					0: {0xd9, 0xf8, 0x70, 0xb0, 0x6d, 0x46, 0x43, 0xc5, 0x9f, 0xbd, 0x0a, 0x9a, 0xd1, 0xe5, 0x5c, 0x43, 0x98, 0xdd, 0xae, 0xf1, 0xca, 0xc2, 0xd7, 0xfb, 0xcf, 0xd5, 0xe0, 0x11, 0xb6, 0x83, 0xb8, 0x33},
					1: {0x50, 0x26, 0x55, 0x2e, 0x7b, 0x21, 0xca, 0xb5, 0x27, 0xe4, 0x16, 0x9e, 0x66, 0x46, 0x02, 0xb8, 0x5d, 0x03, 0x67, 0x0b, 0xb5, 0x57, 0xe3, 0x29, 0x18, 0xd9, 0x33, 0xe3, 0xd5, 0x92, 0x5c, 0x7e},
				},
			},
			lastOutputTime:        time.Time{},
			nextOutputTime:        time.Unix(0, 10000),
			finalizingBlockHeight: 0,

			expected: func() (storageRoot []byte, lastOutputTime time.Time, nextOutputTime time.Time, finalizingBlockHeight int64) {
				return []byte{0xff, 0xd4, 0x7a, 0x71, 0xf6, 0x3a, 0x8a, 0x50, 0x09, 0x56, 0xef, 0x34, 0xb1, 0xfa, 0xbb, 0xd4, 0x2f, 0x07, 0xc8, 0x5e, 0x77, 0xf7, 0xad, 0x21, 0x27, 0x01, 0xe0, 0x64, 0xda, 0xbd, 0xf6, 0xa3},
					time.Unix(0, 10100), time.Unix(0, 10300), 0
			},
			expectedStage: []types.KV{
				{
					Key:   append([]byte("/test_child/working_tree/"), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05}...),
					Value: []byte(`{"version":5,"index":3,"leaf_count":3,"start_leaf_index":10,"last_siblings":{"0":"2fhwsG1GQ8WfvQqa0eVcQ5jdrvHKwtf7z9XgEbaDuDM=","1":"rRHIp/aKAeTbiJgLTE+o5pTqhf9HmGTslmATJK72mmc=","2":"/9R6cfY6ilAJVu80sfq71C8HyF53960hJwHgZNq99qM="},"done":true}`),
				},
				{
					Key:   append([]byte("/test_child/finalized_tree/"), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0a}...),
					Value: []byte(`{"tree_index":3,"tree_height":2,"root":"/9R6cfY6ilAJVu80sfq71C8HyF53960hJwHgZNq99qM=","start_leaf_index":10,"leaf_count":3,"extra_data":"eyJibG9ja19udW1iZXIiOjUsImJsb2NrX2hhc2giOiJkR1Z6ZEY5aWJHOWphMTlwWkE9PSJ9"}`),
				},
				{ // height 0, index 3
					Key:   append([]byte("/test_child/node/"), []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3}...),
					Value: []byte{0xd9, 0xf8, 0x70, 0xb0, 0x6d, 0x46, 0x43, 0xc5, 0x9f, 0xbd, 0x0a, 0x9a, 0xd1, 0xe5, 0x5c, 0x43, 0x98, 0xdd, 0xae, 0xf1, 0xca, 0xc2, 0xd7, 0xfb, 0xcf, 0xd5, 0xe0, 0x11, 0xb6, 0x83, 0xb8, 0x33},
				},
				{ // height 1, index 1
					Key:   append([]byte("/test_child/node/"), []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1}...),
					Value: []byte{0xad, 0x11, 0xc8, 0xa7, 0xf6, 0x8a, 0x01, 0xe4, 0xdb, 0x88, 0x98, 0x0b, 0x4c, 0x4f, 0xa8, 0xe6, 0x94, 0xea, 0x85, 0xff, 0x47, 0x98, 0x64, 0xec, 0x96, 0x60, 0x13, 0x24, 0xae, 0xf6, 0x9a, 0x67},
				},
				{ // height 2, index 0
					Key:   append([]byte("/test_child/node/"), []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3, 0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}...),
					Value: []byte{0xff, 0xd4, 0x7a, 0x71, 0xf6, 0x3a, 0x8a, 0x50, 0x09, 0x56, 0xef, 0x34, 0xb1, 0xfa, 0xbb, 0xd4, 0x2f, 0x07, 0xc8, 0x5e, 0x77, 0xf7, 0xad, 0x21, 0x27, 0x01, 0xe0, 0x64, 0xda, 0xbd, 0xf6, 0xa3},
				},
			},
			err:   false,
			panic: false,
		},
		{
			name:         "passed finalizing block height",
			blockHeight:  10,
			latestHeight: 10,
			blockHeader: cmtproto.Header{
				Time: time.Unix(0, 10100),
			},
			lastWorkingTree: merkletypes.TreeInfo{
				Version:        9,
				Index:          3,
				LeafCount:      3,
				StartLeafIndex: 10,
				LastSiblings:   map[uint8][]byte{},
			},
			finalizingBlockHeight: 5,

			expected:      nil,
			expectedStage: nil,
			err:           false,
			panic:         true,
		},
		{ //nolint
			name:         "output time not reached",
			blockHeight:  5,
			latestHeight: 5,
			blockHeader: cmtproto.Header{
				Time: time.Unix(0, 9900),
			},
			lastWorkingTree: merkletypes.TreeInfo{
				Version:        4,
				Index:          3,
				LeafCount:      3,
				StartLeafIndex: 10,
				LastSiblings: map[uint8][]byte{
					0: {0xd9, 0xf8, 0x70, 0xb0, 0x6d, 0x46, 0x43, 0xc5, 0x9f, 0xbd, 0x0a, 0x9a, 0xd1, 0xe5, 0x5c, 0x43, 0x98, 0xdd, 0xae, 0xf1, 0xca, 0xc2, 0xd7, 0xfb, 0xcf, 0xd5, 0xe0, 0x11, 0xb6, 0x83, 0xb8, 0x33},
					1: {0x50, 0x26, 0x55, 0x2e, 0x7b, 0x21, 0xca, 0xb5, 0x27, 0xe4, 0x16, 0x9e, 0x66, 0x46, 0x02, 0xb8, 0x5d, 0x03, 0x67, 0x0b, 0xb5, 0x57, 0xe3, 0x29, 0x18, 0xd9, 0x33, 0xe3, 0xd5, 0x92, 0x5c, 0x7e},
				},
			},
			lastOutputTime:        time.Time{},
			nextOutputTime:        time.Unix(0, 10000),
			finalizingBlockHeight: 0,

			expected: nil,
			expectedStage: []types.KV{
				{
					Key:   append([]byte("/test_child/working_tree/"), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05}...),
					Value: []byte(`{"version":5,"index":3,"leaf_count":3,"start_leaf_index":10,"last_siblings":{"0":"2fhwsG1GQ8WfvQqa0eVcQ5jdrvHKwtf7z9XgEbaDuDM=","1":"UCZVLnshyrUn5BaeZkYCuF0DZwu1V+MpGNkz49WSXH4="},"done":false}`),
				},
			},
			err:   false,
			panic: false,
		},
		{ //nolint
			name:         "latest height not reached",
			blockHeight:  5,
			latestHeight: 6,
			blockHeader: cmtproto.Header{
				Time: time.Unix(0, 9900),
			},
			lastWorkingTree: merkletypes.TreeInfo{
				Version:        4,
				Index:          3,
				LeafCount:      3,
				StartLeafIndex: 10,
				LastSiblings: map[uint8][]byte{
					0: {0xd9, 0xf8, 0x70, 0xb0, 0x6d, 0x46, 0x43, 0xc5, 0x9f, 0xbd, 0x0a, 0x9a, 0xd1, 0xe5, 0x5c, 0x43, 0x98, 0xdd, 0xae, 0xf1, 0xca, 0xc2, 0xd7, 0xfb, 0xcf, 0xd5, 0xe0, 0x11, 0xb6, 0x83, 0xb8, 0x33},
					1: {0x50, 0x26, 0x55, 0x2e, 0x7b, 0x21, 0xca, 0xb5, 0x27, 0xe4, 0x16, 0x9e, 0x66, 0x46, 0x02, 0xb8, 0x5d, 0x03, 0x67, 0x0b, 0xb5, 0x57, 0xe3, 0x29, 0x18, 0xd9, 0x33, 0xe3, 0xd5, 0x92, 0x5c, 0x7e},
				},
			},
			lastOutputTime:        time.Time{},
			nextOutputTime:        time.Unix(0, 10000),
			finalizingBlockHeight: 0,

			expected: nil,
			expectedStage: []types.KV{
				{
					Key:   append([]byte("/test_child/working_tree/"), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05}...),
					Value: []byte(`{"version":5,"index":3,"leaf_count":3,"start_leaf_index":10,"last_siblings":{"0":"2fhwsG1GQ8WfvQqa0eVcQ5jdrvHKwtf7z9XgEbaDuDM=","1":"UCZVLnshyrUn5BaeZkYCuF0DZwu1V+MpGNkz49WSXH4="},"done":false}`),
				},
			},
			err:   false,
			panic: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			basedb, err := db.NewMemDB()
			require.NoError(t, err)

			childdb := basedb.WithPrefix([]byte("test_child"))
			childNode := node.NewTestNode(nodetypes.NodeConfig{}, childdb, nil, nil, nil, nil)

			mk, err := merkle.NewMerkle(ophosttypes.GenerateNodeHash)
			require.NoError(t, err)
			err = mk.PrepareWorkingTree(tc.lastWorkingTree)
			require.NoError(t, err)

			stage := childdb.NewStage().(*db.Stage)
			ch := Child{
				BaseChild: childprovider.NewTestBaseChild(0, childNode, mk, bridgeInfo, nil, nodetypes.NodeConfig{}),
				stage:     stage,

				finalizingBlockHeight: tc.finalizingBlockHeight,
				lastOutputTime:        tc.lastOutputTime,
				nextOutputTime:        tc.nextOutputTime,
			}

			ctx := types.NewContext(context.Background(), zap.NewNop(), "")
			if tc.panic {
				require.Panics(t, func() {
					ch.handleTree(ctx, tc.blockHeight, tc.latestHeight, blockId, tc.blockHeader) //nolint
				})
				return
			}

			storageRoot, err := ch.handleTree(ctx, tc.blockHeight, tc.latestHeight, blockId, tc.blockHeader)
			if !tc.err {
				require.NoError(t, err)

				if tc.expected != nil {
					expectedStorageRoot, expectedLastOutputTime, expectedNextOutputTime, expectedFinalizingBlockHeight := tc.expected()
					require.Equal(t, expectedStorageRoot, storageRoot)
					require.Equal(t, expectedLastOutputTime, ch.lastOutputTime)
					require.Equal(t, expectedNextOutputTime, ch.nextOutputTime)
					require.Equal(t, expectedFinalizingBlockHeight, ch.finalizingBlockHeight)
				}

				if tc.expectedStage != nil {
					allkvs := stage.All()
					for _, kv := range tc.expectedStage {
						require.Equal(t, kv.Value, allkvs[string(kv.Key)])
					}
				}
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestHandleOutput(t *testing.T) {
	cases := []struct {
		name        string
		blockHeight int64
		version     uint8
		blockId     []byte
		outputIndex uint64
		storageRoot []byte
		bridgeInfo  ophosttypes.QueryBridgeResponse
		host        *mockHost
		expected    sdk.Msg
		err         bool
	}{
		{
			name:        "success",
			blockHeight: 10,
			version:     1,
			blockId:     []byte("latestBlockHashlatestBlockHashla"),
			outputIndex: 1,
			storageRoot: []byte("storageRootstorageRootstorageRoo"),
			bridgeInfo:  ophosttypes.QueryBridgeResponse{BridgeId: 1},
			host:        NewMockHost(nil, nil, 1, "sender0", nil),
			expected: &ophosttypes.MsgProposeOutput{
				Proposer:      "sender0",
				BridgeId:      1,
				OutputIndex:   1,
				L2BlockNumber: 10,
				OutputRoot:    []byte{0xc7, 0x4e, 0xaa, 0x00, 0xbb, 0xc8, 0x16, 0xd2, 0x94, 0x39, 0x01, 0x4c, 0xf7, 0x36, 0x3e, 0x29, 0xb1, 0x85, 0x18, 0x8c, 0xd4, 0x6a, 0x38, 0xfd, 0x64, 0x1f, 0xe5, 0x9f, 0xe4, 0x00, 0xbc, 0xf2},
			},
			err: false,
		},
		{
			name:        "host no broadcaster",
			blockHeight: 10,
			version:     1,
			blockId:     []byte("latestBlockHashlatestBlockHashla"),
			outputIndex: 1,
			storageRoot: []byte("storageRootstorageRootstorageRoo"),
			bridgeInfo:  ophosttypes.QueryBridgeResponse{BridgeId: 1},
			host:        NewMockHost(nil, nil, 1, "", nil),
			expected:    nil,
			err:         false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			basedb, err := db.NewMemDB()
			require.NoError(t, err)

			childdb := basedb.WithPrefix([]byte("test_child"))
			childNode := node.NewTestNode(nodetypes.NodeConfig{}, childdb, nil, nil, nil, nil)

			ch := Child{
				BaseChild: childprovider.NewTestBaseChild(0, childNode, nil, tc.bridgeInfo, nil, nodetypes.NodeConfig{}),
				host:      tc.host,
			}

			err = ch.handleOutput(tc.blockHeight, tc.version, tc.blockId, tc.outputIndex, tc.storageRoot)
			if !tc.err {
				require.NoError(t, err)
				msg := ch.GetMsgQueue()
				if tc.expected != nil {
					require.Equal(t, 1, len(msg))
					require.Equal(t, tc.expected, msg[tc.host.baseAccount][0])
				} else {
					require.Empty(t, msg[tc.host.baseAccount])
				}
			} else {
				require.Error(t, err)
			}
		})
	}
}
