package batchsubmitter

import (
	"sync"
	"testing"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/node"
	"github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateBatchInfo(t *testing.T) {
	baseDB, err := db.NewMemDB()
	require.NoError(t, err)

	batchDB := baseDB.WithPrefix([]byte("test_batch"))
	appCodec, txConfig, err := childprovider.GetCodec("init")
	require.NoError(t, err)
	batchNode := node.NewTestNode(types.NodeConfig{}, batchDB, appCodec, txConfig, nil, nil)

	cases := []struct {
		name               string
		existingBatchInfos []ophosttypes.BatchInfoWithOutput
		chain              string
		submitter          string
		outputIndex        uint64
		l2BlockNumber      int64
		expectedBatchInfos []ophosttypes.BatchInfoWithOutput
		panic              bool
	}{
		{
			name: "success",
			existingBatchInfos: []ophosttypes.BatchInfoWithOutput{
				{
					BatchInfo: ophosttypes.BatchInfo{
						ChainType: ophosttypes.BatchInfo_CHAIN_TYPE_INITIA,
						Submitter: "submitter0",
					},
					Output: ophosttypes.Output{
						L2BlockNumber: 100,
					},
				},
			},
			chain:         "INITIA",
			submitter:     "submitter1",
			outputIndex:   1,
			l2BlockNumber: 110,
			expectedBatchInfos: []ophosttypes.BatchInfoWithOutput{
				{
					BatchInfo: ophosttypes.BatchInfo{
						ChainType: ophosttypes.BatchInfo_CHAIN_TYPE_INITIA,
						Submitter: "submitter0",
					},
					Output: ophosttypes.Output{
						L2BlockNumber: 100,
					},
				},
				{
					BatchInfo: ophosttypes.BatchInfo{
						ChainType: ophosttypes.BatchInfo_CHAIN_TYPE_INITIA,
						Submitter: "submitter1",
					},
					Output: ophosttypes.Output{
						L2BlockNumber: 110,
					},
				},
			},
			panic: false,
		},
		{
			name: "past l2 block number",
			existingBatchInfos: []ophosttypes.BatchInfoWithOutput{
				{
					BatchInfo: ophosttypes.BatchInfo{
						ChainType: ophosttypes.BatchInfo_CHAIN_TYPE_INITIA,
						Submitter: "submitter0",
					},
					Output: ophosttypes.Output{
						L2BlockNumber: 100,
					},
				},
			},
			chain:         "INITIA",
			submitter:     "submitter1",
			outputIndex:   1,
			l2BlockNumber: 90,
			expectedBatchInfos: []ophosttypes.BatchInfoWithOutput{
				{
					BatchInfo: ophosttypes.BatchInfo{
						ChainType: ophosttypes.BatchInfo_CHAIN_TYPE_INITIA,
						Submitter: "submitter0",
					},
					Output: ophosttypes.Output{
						L2BlockNumber: 100,
					},
				},
			},
			panic: false,
		},
		{
			name:               "empty batch infos",
			existingBatchInfos: []ophosttypes.BatchInfoWithOutput{},
			chain:              "test_chain",
			submitter:          "test_submitter",
			outputIndex:        1,
			l2BlockNumber:      1,
			expectedBatchInfos: []ophosttypes.BatchInfoWithOutput{},
			panic:              true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			batchSubmitter := BatchSubmitter{
				node:        batchNode,
				batchInfoMu: &sync.Mutex{},
				batchInfos:  tc.existingBatchInfos,
			}

			if tc.panic {
				require.Panics(t, func() {
					batchSubmitter.UpdateBatchInfo(tc.chain, tc.submitter, tc.outputIndex, tc.l2BlockNumber)
				})
				return
			}

			batchSubmitter.UpdateBatchInfo(tc.chain, tc.submitter, tc.outputIndex, tc.l2BlockNumber)
			assert.Equal(t, tc.expectedBatchInfos, batchSubmitter.batchInfos)
		})
	}
}
