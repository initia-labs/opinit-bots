package child

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	"github.com/initia-labs/opinit-bots/db"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/merkle"
	merkletypes "github.com/initia-labs/opinit-bots/merkle/types"
	"github.com/initia-labs/opinit-bots/node"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func uint64Ptr(i uint64) *uint64 {
	return &i
}

func saveQueryData(t *testing.T, db types.DB) {
	for i := 1; i <= 5; i++ {
		withdrawal := executortypes.WithdrawalData{
			Sequence:       uint64(i),
			From:           "from",
			To:             "to",
			Amount:         100,
			BaseDenom:      "baseDenom",
			WithdrawalHash: []byte("withdrawalHash"),
			TxTime:         int64(i),
			TxHeight:       int64(i),
			TxHash:         fmt.Sprintf("txHash%d", i),
		}
		err := SaveWithdrawal(db, withdrawal)
		require.NoError(t, err)
	}

	for i := 7; i <= 11; i++ {
		withdrawal := executortypes.WithdrawalData{
			Sequence:       uint64(i),
			From:           "from",
			To:             "to2",
			Amount:         1000,
			BaseDenom:      "baseDenom",
			WithdrawalHash: []byte("withdrawalHash"),
			TxTime:         int64(i),
			TxHeight:       int64(i),
			TxHash:         fmt.Sprintf("txHash%d", i),
		}
		err := SaveWithdrawal(db, withdrawal)
		require.NoError(t, err)
	}

	extraData := executortypes.NewTreeExtraData(10, 10, []byte("00000000000000000000000blockid10"))
	extraDataBz, err := json.Marshal(extraData)
	require.NoError(t, err)

	err = merkle.SaveFinalizedTree(db, merkletypes.FinalizedTreeInfo{
		TreeIndex:      1,
		TreeHeight:     2,
		Root:           []byte("000000000000000000000000hash1234"),
		StartLeafIndex: 1,
		LeafCount:      4,
		ExtraData:      extraDataBz,
	})
	require.NoError(t, err)
	err = merkle.SaveNodes(db, []merkletypes.Node{
		{TreeIndex: 1, Height: 0, LocalNodeIndex: 0, Data: []byte("000000000000000000000000000hash1")},
		{TreeIndex: 1, Height: 0, LocalNodeIndex: 1, Data: []byte("000000000000000000000000000hash2")},
		{TreeIndex: 1, Height: 0, LocalNodeIndex: 2, Data: []byte("000000000000000000000000000hash3")},
		{TreeIndex: 1, Height: 0, LocalNodeIndex: 3, Data: []byte("000000000000000000000000000hash4")},
		{TreeIndex: 1, Height: 1, LocalNodeIndex: 0, Data: []byte("00000000000000000000000000hash12")},
		{TreeIndex: 1, Height: 1, LocalNodeIndex: 1, Data: []byte("00000000000000000000000000hash34")},
		{TreeIndex: 1, Height: 2, LocalNodeIndex: 0, Data: []byte("000000000000000000000000hash1234")},
	}...)
	require.NoError(t, err)

	extraData = executortypes.NewTreeExtraData(100, 100, []byte("0000000000000000000000blockid100"))
	extraDataBz, err = json.Marshal(extraData)
	require.NoError(t, err)
	err = merkle.SaveFinalizedTree(db, merkletypes.FinalizedTreeInfo{
		TreeIndex:      2,
		TreeHeight:     3,
		Root:           []byte("00000000000000000000hash56789999"),
		StartLeafIndex: 5,
		LeafCount:      5,
		ExtraData:      extraDataBz,
	})
	require.NoError(t, err)
	err = merkle.SaveNodes(db, []merkletypes.Node{
		{TreeIndex: 2, Height: 0, LocalNodeIndex: 0, Data: []byte("000000000000000000000000000hash5")},
		{TreeIndex: 2, Height: 0, LocalNodeIndex: 1, Data: []byte("000000000000000000000000000hash6")},
		{TreeIndex: 2, Height: 0, LocalNodeIndex: 2, Data: []byte("000000000000000000000000000hash7")},
		{TreeIndex: 2, Height: 0, LocalNodeIndex: 3, Data: []byte("000000000000000000000000000hash8")},
		{TreeIndex: 2, Height: 0, LocalNodeIndex: 4, Data: []byte("000000000000000000000000000hash9")},
		{TreeIndex: 2, Height: 0, LocalNodeIndex: 5, Data: []byte("000000000000000000000000000hash9")},
		{TreeIndex: 2, Height: 0, LocalNodeIndex: 6, Data: []byte("000000000000000000000000000hash9")},
		{TreeIndex: 2, Height: 0, LocalNodeIndex: 7, Data: []byte("000000000000000000000000000hash9")},
		{TreeIndex: 2, Height: 1, LocalNodeIndex: 0, Data: []byte("00000000000000000000000000hash56")},
		{TreeIndex: 2, Height: 1, LocalNodeIndex: 1, Data: []byte("00000000000000000000000000hash78")},
		{TreeIndex: 2, Height: 1, LocalNodeIndex: 2, Data: []byte("00000000000000000000000000hash99")},
		{TreeIndex: 2, Height: 1, LocalNodeIndex: 3, Data: []byte("00000000000000000000000000hash99")},
		{TreeIndex: 2, Height: 2, LocalNodeIndex: 0, Data: []byte("000000000000000000000000hash5678")},
		{TreeIndex: 2, Height: 2, LocalNodeIndex: 1, Data: []byte("000000000000000000000000hash9999")},
		{TreeIndex: 2, Height: 3, LocalNodeIndex: 0, Data: []byte("00000000000000000000hash56789999")},
	}...)
	require.NoError(t, err)

	extraData = executortypes.NewTreeExtraData(1000, 1000, []byte("000000000000000000000blockid1000"))
	extraDataBz, err = json.Marshal(extraData)
	require.NoError(t, err)
	err = merkle.SaveFinalizedTree(db, merkletypes.FinalizedTreeInfo{
		TreeIndex:      3,
		TreeHeight:     1,
		Root:           []byte("000000000000000000000000hash1010"),
		StartLeafIndex: 10,
		LeafCount:      1,
		ExtraData:      extraDataBz,
	})
	require.NoError(t, err)
	err = merkle.SaveNodes(db, []merkletypes.Node{
		{TreeIndex: 3, Height: 0, LocalNodeIndex: 0, Data: []byte("00000000000000000000000000hash10")},
		{TreeIndex: 3, Height: 0, LocalNodeIndex: 1, Data: []byte("00000000000000000000000000hash10")},
		{TreeIndex: 3, Height: 1, LocalNodeIndex: 0, Data: []byte("000000000000000000000000hash1010")},
	}...)
	require.NoError(t, err)
}

func TestQueryWithdrawal(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)
	childDB := db.WithPrefix([]byte("test_child"))
	childNode := node.NewTestNode(nodetypes.NodeConfig{}, childDB, nil, nil, nil, nil)

	ch := Child{
		BaseChild: childprovider.NewTestBaseChild(0, childNode, nil, opchildtypes.BridgeInfo{
			BridgeId: 1,
		}, nil, nodetypes.NodeConfig{}),
	}

	saveQueryData(t, childDB)

	cases := []struct {
		name     string
		sequence uint64
		result   executortypes.QueryWithdrawalResponse
		expected bool
	}{
		{
			name:     "1",
			sequence: 1,
			result: executortypes.QueryWithdrawalResponse{
				Sequence:    1,
				From:        "from",
				To:          "to",
				Amount:      sdk.NewInt64Coin("baseDenom", 100),
				OutputIndex: 1,
				BridgeId:    1,
				WithdrawalProofs: [][]byte{
					[]byte("000000000000000000000000000hash2"),
					[]byte("00000000000000000000000000hash34"),
				},
				Version:       []byte{0},
				StorageRoot:   []byte("000000000000000000000000hash1234"),
				LastBlockHash: []byte("00000000000000000000000blockid10"),
				TxTime:        time.Unix(0, 1),
				TxHeight:      1,
				TxHash:        "txHash1",
			},
			expected: true,
		},
		{
			name:     "5",
			sequence: 5,
			result: executortypes.QueryWithdrawalResponse{
				Sequence:    5,
				From:        "from",
				To:          "to",
				Amount:      sdk.NewInt64Coin("baseDenom", 100),
				OutputIndex: 2,
				BridgeId:    1,
				WithdrawalProofs: [][]byte{
					[]byte("000000000000000000000000000hash6"),
					[]byte("00000000000000000000000000hash78"),
					[]byte("000000000000000000000000hash9999"),
				},
				Version:       []byte{0},
				StorageRoot:   []byte("00000000000000000000hash56789999"),
				LastBlockHash: []byte("0000000000000000000000blockid100"),
				TxTime:        time.Unix(0, 5),
				TxHeight:      5,
				TxHash:        "txHash5",
			},
			expected: true,
		},
		{
			name:     "not existing sequence 6",
			sequence: 6,
			result:   executortypes.QueryWithdrawalResponse{},
			expected: false,
		},
		{
			name:     "7",
			sequence: 7,
			result: executortypes.QueryWithdrawalResponse{
				Sequence:    7,
				From:        "from",
				To:          "to2",
				Amount:      sdk.NewInt64Coin("baseDenom", 1000),
				OutputIndex: 2,
				BridgeId:    1,
				WithdrawalProofs: [][]byte{
					[]byte("000000000000000000000000000hash8"),
					[]byte("00000000000000000000000000hash56"),
					[]byte("000000000000000000000000hash9999"),
				},
				Version:       []byte{0},
				StorageRoot:   []byte("00000000000000000000hash56789999"),
				LastBlockHash: []byte("0000000000000000000000blockid100"),
				TxTime:        time.Unix(0, 7),
				TxHeight:      7,
				TxHash:        "txHash7",
			},
			expected: true,
		},
		{
			name:     "not finalized sequence 11",
			sequence: 11,
			result: executortypes.QueryWithdrawalResponse{
				Sequence:         11,
				From:             "from",
				To:               "to2",
				Amount:           sdk.NewInt64Coin("baseDenom", 1000),
				OutputIndex:      0,
				BridgeId:         1,
				WithdrawalProofs: nil,
				Version:          []byte{0},
				StorageRoot:      nil,
				LastBlockHash:    nil,
				TxTime:           time.Unix(0, 11),
				TxHeight:         11,
				TxHash:           "txHash11",
			},
			expected: true,
		},
		{
			name:     "not existing sequence 0",
			sequence: 0,
			result:   executortypes.QueryWithdrawalResponse{},
			expected: false,
		},
		{
			name:     "not existing sequence 12",
			sequence: 12,
			result:   executortypes.QueryWithdrawalResponse{},
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ch.QueryWithdrawal(tc.sequence)
			if tc.expected {
				require.NoError(t, err)
				require.Equal(t, tc.result, result)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestQueryWithdrawals(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)
	childDB := db.WithPrefix([]byte("test_child"))
	childNode := node.NewTestNode(nodetypes.NodeConfig{}, childDB, nil, nil, nil, nil)

	ch := Child{
		BaseChild: childprovider.NewTestBaseChild(0, childNode, nil, opchildtypes.BridgeInfo{
			BridgeId: 1,
		}, nil, nodetypes.NodeConfig{}),
	}

	saveQueryData(t, childDB)

	cases := []struct {
		name      string
		address   string
		offset    uint64
		limit     uint64
		descOrder bool
		result    executortypes.QueryWithdrawalsResponse
		expected  bool
	}{
		{
			name:      "to, offset 0, limit 3, asc",
			address:   "to",
			offset:    0,
			limit:     3,
			descOrder: false,
			result: executortypes.QueryWithdrawalsResponse{
				Withdrawals: []executortypes.QueryWithdrawalResponse{
					{
						Sequence:    1,
						From:        "from",
						To:          "to",
						Amount:      sdk.NewInt64Coin("baseDenom", 100),
						OutputIndex: 1,
						BridgeId:    1,
						WithdrawalProofs: [][]byte{
							[]byte("000000000000000000000000000hash2"),
							[]byte("00000000000000000000000000hash34"),
						},
						Version:       []byte{0},
						StorageRoot:   []byte("000000000000000000000000hash1234"),
						LastBlockHash: []byte("00000000000000000000000blockid10"),
						TxTime:        time.Unix(0, 1),
						TxHeight:      1,
						TxHash:        "txHash1",
					},
					{
						Sequence:    2,
						From:        "from",
						To:          "to",
						Amount:      sdk.NewInt64Coin("baseDenom", 100),
						OutputIndex: 1,
						BridgeId:    1,
						WithdrawalProofs: [][]byte{
							[]byte("000000000000000000000000000hash1"),
							[]byte("00000000000000000000000000hash34"),
						},
						Version:       []byte{0},
						StorageRoot:   []byte("000000000000000000000000hash1234"),
						LastBlockHash: []byte("00000000000000000000000blockid10"),
						TxTime:        time.Unix(0, 2),
						TxHeight:      2,
						TxHash:        "txHash2",
					},
					{
						Sequence:    3,
						From:        "from",
						To:          "to",
						Amount:      sdk.NewInt64Coin("baseDenom", 100),
						OutputIndex: 1,
						BridgeId:    1,
						WithdrawalProofs: [][]byte{
							[]byte("000000000000000000000000000hash4"),
							[]byte("00000000000000000000000000hash12"),
						},
						Version:       []byte{0},
						StorageRoot:   []byte("000000000000000000000000hash1234"),
						LastBlockHash: []byte("00000000000000000000000blockid10"),
						TxTime:        time.Unix(0, 3),
						TxHeight:      3,
						TxHash:        "txHash3",
					},
				},
				Next: uint64Ptr(4),
			},
			expected: true,
		},
		{
			name:      "to, offset 0, limit 0, desc",
			address:   "to",
			offset:    0,
			limit:     0,
			descOrder: true,
			result: executortypes.QueryWithdrawalsResponse{
				Withdrawals: []executortypes.QueryWithdrawalResponse{},
				Next:        uint64Ptr(5),
			},
			expected: true,
		},
		{
			name:      "to, offset 1, limit 5, desc",
			address:   "to",
			offset:    1,
			limit:     5,
			descOrder: true,
			result: executortypes.QueryWithdrawalsResponse{
				Withdrawals: []executortypes.QueryWithdrawalResponse{
					{
						Sequence:    1,
						From:        "from",
						To:          "to",
						Amount:      sdk.NewInt64Coin("baseDenom", 100),
						OutputIndex: 1,
						BridgeId:    1,
						WithdrawalProofs: [][]byte{
							[]byte("000000000000000000000000000hash2"),
							[]byte("00000000000000000000000000hash34"),
						},
						Version:       []byte{0},
						StorageRoot:   []byte("000000000000000000000000hash1234"),
						LastBlockHash: []byte("00000000000000000000000blockid10"),
						TxTime:        time.Unix(0, 1),
						TxHeight:      1,
						TxHash:        "txHash1",
					},
				},
			},
			expected: true,
		},
		{
			name:      "to2, offset 11, limit 10, desc",
			address:   "to2",
			offset:    11,
			limit:     10,
			descOrder: true,
			result: executortypes.QueryWithdrawalsResponse{
				Withdrawals: []executortypes.QueryWithdrawalResponse{
					{
						Sequence:         11,
						From:             "from",
						To:               "to2",
						Amount:           sdk.NewInt64Coin("baseDenom", 1000),
						OutputIndex:      0,
						BridgeId:         1,
						WithdrawalProofs: nil,
						Version:          []byte{0},
						StorageRoot:      nil,
						LastBlockHash:    nil,
						TxTime:           time.Unix(0, 11),
						TxHeight:         11,
						TxHash:           "txHash11",
					},
					{
						Sequence:    10,
						From:        "from",
						To:          "to2",
						Amount:      sdk.NewInt64Coin("baseDenom", 1000),
						OutputIndex: 3,
						BridgeId:    1,
						WithdrawalProofs: [][]byte{
							[]byte("00000000000000000000000000hash10"),
						},
						Version:       []byte{0},
						StorageRoot:   []byte("000000000000000000000000hash1010"),
						LastBlockHash: []byte("000000000000000000000blockid1000"),
						TxTime:        time.Unix(0, 10),
						TxHeight:      10,
						TxHash:        "txHash10",
					},
					{
						Sequence:    9,
						From:        "from",
						To:          "to2",
						Amount:      sdk.NewInt64Coin("baseDenom", 1000),
						OutputIndex: 2,
						BridgeId:    1,
						WithdrawalProofs: [][]byte{
							[]byte("000000000000000000000000000hash9"),
							[]byte("00000000000000000000000000hash99"),
							[]byte("000000000000000000000000hash5678"),
						},
						Version:       []byte{0},
						StorageRoot:   []byte("00000000000000000000hash56789999"),
						LastBlockHash: []byte("0000000000000000000000blockid100"),
						TxTime:        time.Unix(0, 9),
						TxHeight:      9,
						TxHash:        "txHash9",
					},
					{
						Sequence:    8,
						From:        "from",
						To:          "to2",
						Amount:      sdk.NewInt64Coin("baseDenom", 1000),
						OutputIndex: 2,
						BridgeId:    1,
						WithdrawalProofs: [][]byte{
							[]byte("000000000000000000000000000hash7"),
							[]byte("00000000000000000000000000hash56"),
							[]byte("000000000000000000000000hash9999"),
						},
						Version:       []byte{0},
						StorageRoot:   []byte("00000000000000000000hash56789999"),
						LastBlockHash: []byte("0000000000000000000000blockid100"),
						TxTime:        time.Unix(0, 8),
						TxHeight:      8,
						TxHash:        "txHash8",
					},
					{
						Sequence:    7,
						From:        "from",
						To:          "to2",
						Amount:      sdk.NewInt64Coin("baseDenom", 1000),
						OutputIndex: 2,
						BridgeId:    1,
						WithdrawalProofs: [][]byte{
							[]byte("000000000000000000000000000hash8"),
							[]byte("00000000000000000000000000hash56"),
							[]byte("000000000000000000000000hash9999"),
						},
						Version:       []byte{0},
						StorageRoot:   []byte("00000000000000000000hash56789999"),
						LastBlockHash: []byte("0000000000000000000000blockid100"),
						TxTime:        time.Unix(0, 7),
						TxHeight:      7,
						TxHash:        "txHash7",
					},
				},
			},
			expected: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ch.QueryWithdrawals(tc.address, tc.offset, tc.limit, tc.descOrder)
			if tc.expected {
				require.NoError(t, err)
				require.Equal(t, tc.result, result)
			} else {
				require.Error(t, err)
			}
		})
	}
}
