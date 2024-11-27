package child

import (
	"context"
	"strconv"
	"testing"
	"time"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/merkle"
	merkletypes "github.com/initia-labs/opinit-bots/merkle/types"
	"github.com/initia-labs/opinit-bots/node"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
		workingTree      merkletypes.TreeInfo
		eventHandlerArgs nodetypes.EventHandlerArgs
		expectedStage    []types.KV
		expectedLog      func() (msg string, fields []zapcore.Field)
		err              bool
		panic            bool
	}{
		{
			name: "success",
			workingTree: merkletypes.TreeInfo{
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
			workingTree: merkletypes.TreeInfo{
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
			workingTree: merkletypes.TreeInfo{
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
			err = mk.PrepareWorkingTree(tc.workingTree)
			require.NoError(t, err)

			stage := childdb.NewStage().(*db.Stage)
			ch := Child{
				BaseChild: childprovider.NewTestBaseChild(0, childNode, mk, bridgeInfo, nil, nodetypes.NodeConfig{}),
				stage:     stage,
			}

			if tc.panic {
				assert.Panics(t, func() {
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
					Value: []byte(`{"index":2,"leaf_count":0,"start_leaf_index":1,"height_data":{},"done":false}`),
				},
			},
			blockHeight:           6,
			initializeTreeFnMaker: nil,
			expected: merkletypes.TreeInfo{
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
					Value: []byte(`{"index":2,"leaf_count":2,"start_leaf_index":1,"height_data":{},"done":true}`),
				},
			},
			blockHeight:           6,
			initializeTreeFnMaker: nil,
			expected: merkletypes.TreeInfo{
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
				assert.Panics(t, func() {
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
		name        string
		bridgeInfo  ophosttypes.QueryBridgeResponse
		hostOutputs map[uint64]ophosttypes.Output
		workingTree merkletypes.TreeInfo
		expected    func() (lastOutputTime time.Time, nextOutputTime time.Time, finalizingBlockHeight int64)
		err         bool
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
			workingTree: merkletypes.TreeInfo{
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
			workingTree: merkletypes.TreeInfo{
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
			workingTree: merkletypes.TreeInfo{
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
			workingTree: merkletypes.TreeInfo{
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
			err = mk.PrepareWorkingTree(tc.workingTree)
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
