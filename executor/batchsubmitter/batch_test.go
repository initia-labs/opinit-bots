package batchsubmitter

import (
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	cmttypes "github.com/cometbft/cometbft/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	client "github.com/initia-labs/opinit-bots/client"
	mockclient "github.com/initia-labs/opinit-bots/client/mock"
	"github.com/initia-labs/opinit-bots/db"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/node"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots/node/rpcclient"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestPrepareBatch(t *testing.T) {
	appCodec, txConfig, err := childprovider.GetCodec("init")
	require.NoError(t, err)

	cases := []struct {
		name                   string
		existingLocalBatchInfo executortypes.LocalBatchInfo
		batchInfoQueue         []ophosttypes.BatchInfoWithOutput
		blockHeight            int64
		expectedLocalBatchInfo executortypes.LocalBatchInfo
		expectedChanges        []types.KV
		err                    bool
		panic                  bool
	}{
		{
			name: "not finalized batch info",
			existingLocalBatchInfo: executortypes.LocalBatchInfo{
				Start:              2,
				End:                0,
				LastSubmissionTime: time.Time{},
				BatchSize:          100,
			},
			batchInfoQueue: []ophosttypes.BatchInfoWithOutput{
				{
					BatchInfo: ophosttypes.BatchInfo{
						ChainType: ophosttypes.BatchInfo_INITIA,
						Submitter: "submitter0",
					},
				},
			},
			blockHeight: 110,
			expectedLocalBatchInfo: executortypes.LocalBatchInfo{
				Start:              2,
				End:                0,
				LastSubmissionTime: time.Time{},
				BatchSize:          100,
			},
			expectedChanges: []types.KV{},
			err:             false,
			panic:           false,
		},
		{
			name: "finalized batch info",
			existingLocalBatchInfo: executortypes.LocalBatchInfo{
				Start:              1,
				End:                100,
				LastSubmissionTime: time.Unix(0, 10000).UTC(),
				BatchSize:          100,
			},
			batchInfoQueue: []ophosttypes.BatchInfoWithOutput{
				{
					BatchInfo: ophosttypes.BatchInfo{
						ChainType: ophosttypes.BatchInfo_INITIA,
						Submitter: "submitter0",
					},
				},
			},
			blockHeight: 101,
			expectedLocalBatchInfo: executortypes.LocalBatchInfo{
				Start:              101,
				End:                0,
				LastSubmissionTime: time.Unix(0, 10000).UTC(),
				BatchSize:          0,
			},
			expectedChanges: []types.KV{},
			err:             false,
			panic:           false,
		},
		{
			name: "existing next batch info, not reached to the l2 block number of the next batch info",
			existingLocalBatchInfo: executortypes.LocalBatchInfo{
				Start:              1,
				End:                0,
				LastSubmissionTime: time.Time{},
				BatchSize:          100,
			},
			batchInfoQueue: []ophosttypes.BatchInfoWithOutput{
				{
					BatchInfo: ophosttypes.BatchInfo{
						ChainType: ophosttypes.BatchInfo_INITIA,
						Submitter: "submitter0",
					},
				},
				{
					BatchInfo: ophosttypes.BatchInfo{
						ChainType: ophosttypes.BatchInfo_CELESTIA,
						Submitter: "submitter0",
					},
					Output: ophosttypes.Output{
						L2BlockNumber: 200,
					},
				},
			},
			blockHeight: 101,
			expectedLocalBatchInfo: executortypes.LocalBatchInfo{
				Start:              1,
				End:                0,
				LastSubmissionTime: time.Time{},
				BatchSize:          100,
			},
			expectedChanges: []types.KV{},
			err:             false,
			panic:           false,
		},
		{
			name: "existing next batch info, reached to the l2 block number of the next batch info",
			existingLocalBatchInfo: executortypes.LocalBatchInfo{
				Start:              51,
				End:                0,
				LastSubmissionTime: time.Unix(0, 10000).UTC(),
				BatchSize:          100,
			},
			batchInfoQueue: []ophosttypes.BatchInfoWithOutput{
				{
					BatchInfo: ophosttypes.BatchInfo{
						ChainType: ophosttypes.BatchInfo_INITIA,
						Submitter: "submitter0",
					},
				},
				{
					BatchInfo: ophosttypes.BatchInfo{
						ChainType: ophosttypes.BatchInfo_CELESTIA,
						Submitter: "submitter0",
					},
					Output: ophosttypes.Output{
						L2BlockNumber: 40,
					},
				},
			},
			blockHeight: 110,
			expectedLocalBatchInfo: executortypes.LocalBatchInfo{
				Start:              41,
				End:                0,
				LastSubmissionTime: time.Unix(0, 10000).UTC(),
				BatchSize:          0,
			},
			expectedChanges: []types.KV{
				{
					Key:   []byte("test_batch/local_batch_info"),
					Value: []byte(`{"start":41,"end":0,"last_submission_time":"1970-01-01T00:00:00.00001Z","batch_size":0}`),
				},
				{
					Key:   []byte("test_batch/synced_height"),
					Value: []byte("40"),
				},
			},
			err:   false,
			panic: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			baseDB, err := db.NewMemDB()
			require.NoError(t, err)

			batchDB := baseDB.WithPrefix([]byte("test_batch"))
			batchNode := node.NewTestNode(nodetypes.NodeConfig{}, batchDB, appCodec, txConfig, nil, nil)

			batchSubmitter := BatchSubmitter{
				node:        batchNode,
				batchInfoMu: &sync.Mutex{},
				batchInfos:  tc.batchInfoQueue,
				da:          NewMockDA(nil, nil, 1, ""),
			}

			err = SaveLocalBatchInfo(batchDB, tc.existingLocalBatchInfo)
			require.NoError(t, err)

			batchSubmitter.batchFile, err = os.CreateTemp("", "batchfile")
			require.NoError(t, err)
			defer os.Remove(batchSubmitter.batchFile.Name())

			batchSubmitter.batchWriter, err = gzip.NewWriterLevel(batchSubmitter.batchFile, 6)
			require.NoError(t, err)
			defer batchSubmitter.batchWriter.Close()

			ctx := types.NewContext(context.Background(), zap.NewNop(), "")
			if tc.panic {
				require.Panics(t, func() {
					batchSubmitter.prepareBatch(ctx, tc.blockHeight) //nolint:errcheck
				})
				for _, expectedKV := range tc.expectedChanges {
					value, err := baseDB.Get(expectedKV.Key)
					require.NoError(t, err)
					require.Equal(t, expectedKV.Value, value)
				}
				require.Equal(t, tc.expectedLocalBatchInfo, *batchSubmitter.localBatchInfo)
				fileSize, err := batchSubmitter.batchFileSize(true)
				require.NoError(t, err)
				require.Equal(t, int64(0), fileSize)
			} else {
				err := batchSubmitter.prepareBatch(ctx, tc.blockHeight)
				if tc.err {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					for _, expectedKV := range tc.expectedChanges {
						value, err := baseDB.Get(expectedKV.Key)
						require.NoError(t, err)
						require.Equal(t, expectedKV.Value, value)
					}
					require.Equal(t, tc.expectedLocalBatchInfo, *batchSubmitter.localBatchInfo)
				}
			}
		})
	}
}

func TestHandleBatch(t *testing.T) {
	var err error

	batchSubmitter := BatchSubmitter{}
	batchSubmitter.batchFile, err = os.CreateTemp("", "batchfile")
	require.NoError(t, err)
	defer os.Remove(batchSubmitter.batchFile.Name())

	batchSubmitter.batchWriter, err = gzip.NewWriterLevel(batchSubmitter.batchFile, 6)
	require.NoError(t, err)
	defer batchSubmitter.batchWriter.Close()

	blockBytes := []byte("block_bytes")
	n, err := batchSubmitter.handleBatch(blockBytes)
	require.NoError(t, err)

	require.Equal(t, n, 11+8) // len(block_bytes) + len(length_prefix)

	blockBytes = []byte("")
	_, err = batchSubmitter.handleBatch(blockBytes)
	require.Error(t, err)
}

func TestFinalizeBatch(t *testing.T) {
	baseDB, err := db.NewMemDB()
	require.NoError(t, err)

	batchDB := baseDB.WithPrefix([]byte("test_batch"))
	daDB := baseDB.WithPrefix([]byte("test_da"))

	appCodec, txConfig, err := childprovider.GetCodec("init")
	require.NoError(t, err)

	mockCaller := mockclient.NewMockCaller()
	testLogger, _ := zap.NewDevelopment()
	rpcClient, err := rpcclient.NewRPCClientWithClient(appCodec, client.NewWithCaller(mockCaller), []string{"http://localhost:26657"}, testLogger)
	require.NoError(t, err)
	batchNode := node.NewTestNode(nodetypes.NodeConfig{}, batchDB, appCodec, txConfig, rpcClient, nil)

	hostCdc, _, err := hostprovider.GetCodec("init")
	require.NoError(t, err)

	mockDA := NewMockDA(daDB, hostCdc, 1, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0")

	batchConfig := executortypes.BatchConfig{
		MaxChunks:    100,
		MaxChunkSize: 10,
	}

	batchSubmitter := BatchSubmitter{
		node:     batchNode,
		da:       mockDA,
		batchCfg: batchConfig,
		localBatchInfo: &executortypes.LocalBatchInfo{
			Start: 1,
			End:   10,
		},
		processedMsgs: make([]btypes.ProcessedMsgs, 0),
	}
	batchSubmitter.batchFile, err = os.CreateTemp("", "batchfile")
	require.NoError(t, err)
	defer os.Remove(batchSubmitter.batchFile.Name())

	batchSubmitter.batchWriter, err = gzip.NewWriterLevel(batchSubmitter.batchFile, 6)
	require.NoError(t, err)
	defer batchSubmitter.batchWriter.Close()

	mockCaller.SetRawCommit(10, []byte("commit_bytes"))

	logger, observedLogs := logCapturer()
	ctx := types.NewContext(context.TODO(), logger, "")

	for i := 0; i < 10; i++ {
		_, err := batchSubmitter.batchWriter.Write([]byte(fmt.Sprintf("block_bytes%d", i)))
		if err != nil {
			require.NoError(t, err)
		}
	}

	mockCount := int64(0)
	mockTimestampFetcher := func() int64 {
		mockCount++
		return mockCount
	}
	types.CurrentNanoTimestamp = mockTimestampFetcher

	err = batchSubmitter.finalizeBatch(ctx, 10)
	require.NoError(t, err)

	logs := observedLogs.TakeAll()
	require.Len(t, logs, 1)

	require.Equal(t, "finalize batch", logs[0].Message)
	require.Equal(t, []zapcore.Field{
		zap.Int64("height", 10),
		zap.Int64("batch start", 1),
		zap.Int64("batch end", 10),
		zap.Int64("batch size", 73),
		zap.Int("chunks", 8),
		zap.Int("txs", 9),
	}, logs[0].Context)

	require.Len(t, batchSubmitter.processedMsgs, 9)

	require.Equal(t, btypes.ProcessedMsgs{
		Sender: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
		Msgs: []sdk.Msg{
			&ophosttypes.MsgRecordBatch{
				Submitter: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
				BridgeId:  1,
				BatchBytes: []byte{
					0x00,                                           // type header
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // start 1
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0a, // end 10
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, // chunks length 8
					0x21, 0x7f, 0xeb, 0x1e, 0x74, 0x90, 0x01, 0x5d, 0xd0, 0xa2, 0xb2, 0x31, 0xb9, 0xce, 0xa4, 0x58, 0x04, 0xdf, 0x3d, 0x2a, 0x9b, 0x37, 0x28, 0x7a, 0xc8, 0x61, 0xbb, 0x45, 0xb8, 0xc0, 0xde, 0x55, // checksum[0]
					0x08, 0x95, 0x3b, 0xa5, 0xb6, 0x25, 0xa4, 0x7b, 0x3e, 0xc0, 0xbe, 0xe6, 0x71, 0xab, 0xa1, 0x05, 0x62, 0x01, 0x8a, 0x03, 0xf3, 0xb8, 0xed, 0x84, 0x22, 0xa0, 0x22, 0x78, 0x61, 0xff, 0xdd, 0x7e, // checksum[1]
					0x1c, 0x65, 0xb4, 0x2e, 0x46, 0x1f, 0xb4, 0x37, 0x96, 0xa8, 0x12, 0x2a, 0x7e, 0xf7, 0xaf, 0x4d, 0xa4, 0x15, 0x8c, 0xe7, 0x92, 0x87, 0x2e, 0xbc, 0xc6, 0xd5, 0xe7, 0x61, 0x38, 0x1e, 0x92, 0xea, // checksum[2]
					0xad, 0x11, 0x06, 0x84, 0x21, 0x2f, 0xd0, 0x05, 0xe7, 0xaa, 0xc9, 0x04, 0x22, 0x7e, 0x1b, 0xc3, 0x04, 0x57, 0x52, 0x64, 0x34, 0xdf, 0x02, 0x56, 0x8d, 0x60, 0x09, 0x4b, 0xa9, 0x5e, 0x3c, 0x70, // checksum[3]
					0xb0, 0x60, 0x0f, 0x49, 0xee, 0x80, 0xc2, 0x63, 0xec, 0xc4, 0x4e, 0xfa, 0x76, 0x2f, 0x87, 0xde, 0x40, 0x6c, 0xd5, 0x4e, 0x68, 0x17, 0xbc, 0x6c, 0x46, 0x1c, 0x74, 0x44, 0x4f, 0xc2, 0x15, 0xb8, // checksum[4]
					0xbe, 0x47, 0xc7, 0x55, 0xab, 0xe2, 0x47, 0x27, 0x22, 0x01, 0xc0, 0xc4, 0xbd, 0xe3, 0xe6, 0xb8, 0xf3, 0x84, 0x13, 0x51, 0x71, 0x10, 0x99, 0x96, 0xab, 0x05, 0x19, 0xca, 0xa0, 0x0e, 0x47, 0xfc, // checksum[5]
					0x18, 0x1b, 0xef, 0x86, 0x5a, 0x29, 0x8e, 0xc0, 0xc9, 0xcb, 0xb6, 0xe0, 0x87, 0xe0, 0x3c, 0x7d, 0x5f, 0x0c, 0xf1, 0xa3, 0x32, 0x0e, 0x11, 0xce, 0x3f, 0x27, 0xb2, 0x30, 0x8f, 0x41, 0x72, 0x51, // checksum[6]
					0x70, 0x9e, 0x80, 0xc8, 0x84, 0x87, 0xa2, 0x41, 0x1e, 0x1e, 0xe4, 0xdf, 0xb9, 0xf2, 0x2a, 0x86, 0x14, 0x92, 0xd2, 0x0c, 0x47, 0x65, 0x15, 0x0c, 0x0c, 0x79, 0x4a, 0xbd, 0x70, 0xf8, 0x14, 0x7c, // checksum[7]
				},
			},
		},
		Timestamp: 1,
		Save:      true,
	}, batchSubmitter.processedMsgs[0])

	require.Equal(t, btypes.ProcessedMsgs{
		Sender: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
		Msgs: []sdk.Msg{
			&ophosttypes.MsgRecordBatch{
				Submitter: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
				BridgeId:  1,
				BatchBytes: []byte{
					0x01,                                           // type chunk
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // start 1
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0a, // end 10
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // index 0
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, // chunks length 8
					0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, // chunkdata
				},
			},
		},
		Timestamp: 2,
		Save:      true,
	}, batchSubmitter.processedMsgs[1])

	require.Equal(t, btypes.ProcessedMsgs{
		Sender: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
		Msgs: []sdk.Msg{
			&ophosttypes.MsgRecordBatch{
				Submitter: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
				BridgeId:  1,
				BatchBytes: []byte{
					0x01,                                           // type chunk
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // start 1
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0a, // end 10
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // index 1
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, // chunks length 8
					0x4a, 0xca, 0xc9, 0x4f, 0xce, 0x8e, 0x4f, 0xaa, 0x2c, 0x49, // chunkdata
				},
			},
		},
		Timestamp: 3,
		Save:      true,
	}, batchSubmitter.processedMsgs[2])

	require.Equal(t, btypes.ProcessedMsgs{
		Sender: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
		Msgs: []sdk.Msg{
			&ophosttypes.MsgRecordBatch{
				Submitter: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
				BridgeId:  1,
				BatchBytes: []byte{
					0x01,                                           // type chunk
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // start 1
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0a, // end 10
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, // index 2
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, // chunks length 8
					0x2d, 0x36, 0x40, 0x62, 0x1b, 0x22, 0xb1, 0x8d, 0x90, 0xd8, // chunkdata
				},
			},
		},
		Timestamp: 4,
		Save:      true,
	}, batchSubmitter.processedMsgs[3])

	require.Equal(t, btypes.ProcessedMsgs{
		Sender: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
		Msgs: []sdk.Msg{
			&ophosttypes.MsgRecordBatch{
				Submitter: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
				BridgeId:  1,
				BatchBytes: []byte{
					0x01,                                           // type chunk
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // start 1
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0a, // end 10
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, // index 3
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, // chunks length 8
					0xc6, 0x48, 0x6c, 0x13, 0x24, 0xb6, 0x29, 0x12, 0xdb, 0x0c, // chunkdata
				},
			},
		},
		Timestamp: 5,
		Save:      true,
	}, batchSubmitter.processedMsgs[4])

	require.Equal(t, btypes.ProcessedMsgs{
		Sender: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
		Msgs: []sdk.Msg{
			&ophosttypes.MsgRecordBatch{
				Submitter: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
				BridgeId:  1,
				BatchBytes: []byte{
					0x01,                                           // type chunk
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // start 1
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0a, // end 10
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04, // index 4
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, // chunks length 8
					0x89, 0x6d, 0x8e, 0xc4, 0xb6, 0x40, 0x62, 0x5b, 0xf2, 0x30, // chunkdata
				},
			},
		},
		Timestamp: 6,
		Save:      true,
	}, batchSubmitter.processedMsgs[5])

	require.Equal(t, btypes.ProcessedMsgs{
		Sender: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
		Msgs: []sdk.Msg{
			&ophosttypes.MsgRecordBatch{
				Submitter: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
				BridgeId:  1,
				BatchBytes: []byte{
					0x01,                                           // type chunk
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // start 1
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0a, // end 10
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05, // index 5
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, // chunks length 8
					0x40, 0x40, 0x72, 0x7e, 0x6e, 0x6e, 0x66, 0x09, 0x44, 0x10, // chunkdata
				},
			},
		},
		Timestamp: 7,
		Save:      true,
	}, batchSubmitter.processedMsgs[6])

	require.Equal(t, btypes.ProcessedMsgs{
		Sender: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
		Msgs: []sdk.Msg{
			&ophosttypes.MsgRecordBatch{
				Submitter: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
				BridgeId:  1,
				BatchBytes: []byte{
					0x01,                                           // type chunk
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // start 1
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0a, // end 10
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x06, // index 6
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, // chunks length 8
					0x10, 0x00, 0x00, 0xff, 0xff, 0x92, 0x7b, 0x8a, 0x85, 0x8c, // chunkdata
				},
			},
		},
		Timestamp: 8,
		Save:      true,
	}, batchSubmitter.processedMsgs[7])

	require.Equal(t, btypes.ProcessedMsgs{
		Sender: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
		Msgs: []sdk.Msg{
			&ophosttypes.MsgRecordBatch{
				Submitter: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
				BridgeId:  1,
				BatchBytes: []byte{
					0x01,                                           // type chunk
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // start 1
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0a, // end 10
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, // index 7
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, // chunks length 8
					0x00, 0x00, 0x00, // chunkdata
				},
			},
		},
		Timestamp: 9,
		Save:      true,
	}, batchSubmitter.processedMsgs[8])
}

func TestCheckBatch(t *testing.T) {
	batchSubmitter := BatchSubmitter{
		bridgeInfo: ophosttypes.QueryBridgeResponse{
			BridgeConfig: ophosttypes.BridgeConfig{
				SubmissionInterval: 15000,
			},
		},
	}

	cases := []struct {
		name           string
		localBatchInfo *executortypes.LocalBatchInfo
		batchConfig    executortypes.BatchConfig
		blockHeight    int64
		latestHeight   int64
		blockTime      time.Time
		expected       bool
	}{
		{
			name: "block time >= last submission time + 2/3 interval, block height == latest height",
			localBatchInfo: &executortypes.LocalBatchInfo{
				Start:              1,
				LastSubmissionTime: time.Unix(0, 0).UTC(),
			},
			batchConfig: executortypes.BatchConfig{
				MaxChunks:         100,
				MaxChunkSize:      10,
				MaxSubmissionTime: 1, // time.Second
			},
			blockHeight:  10,
			latestHeight: 10,
			blockTime:    time.Unix(0, 10001).UTC(),
			expected:     true,
		},
		{
			name: "block time >= last submission time + 2/3 interval, block height < latest height",
			localBatchInfo: &executortypes.LocalBatchInfo{
				Start:              1,
				LastSubmissionTime: time.Unix(0, 0),
			},
			batchConfig: executortypes.BatchConfig{
				MaxChunks:         100,
				MaxChunkSize:      10,
				MaxSubmissionTime: 1,
			},
			blockHeight:  10,
			latestHeight: 20,
			blockTime:    time.Unix(0, 10001).UTC(),
			expected:     false,
		},
		{
			name: "block time < last submission time + 2/3 interval, block height == latest height",
			localBatchInfo: &executortypes.LocalBatchInfo{
				Start:              1,
				LastSubmissionTime: time.Unix(0, 10000).UTC(),
			},
			batchConfig: executortypes.BatchConfig{
				MaxChunks:         100,
				MaxChunkSize:      10,
				MaxSubmissionTime: 1,
			},
			blockHeight:  10,
			latestHeight: 10,
			blockTime:    time.Unix(0, 10001).UTC(),
			expected:     false,
		},
		{
			name: "block time > last submission time + max submission time, block height == latest height",
			localBatchInfo: &executortypes.LocalBatchInfo{
				Start:              1,
				LastSubmissionTime: time.Unix(0, 0).UTC(),
			},
			batchConfig: executortypes.BatchConfig{
				MaxChunks:         100,
				MaxChunkSize:      10,
				MaxSubmissionTime: 1,
			},
			blockHeight:  10,
			latestHeight: 10,
			blockTime:    time.Unix(0, 1000*1000*1000+1).UTC(),
			expected:     true,
		},
		{
			name: "block time > last submission time + max submission time, block height != latest height",
			localBatchInfo: &executortypes.LocalBatchInfo{
				Start:              1,
				LastSubmissionTime: time.Unix(0, 0).UTC(),
			},
			batchConfig: executortypes.BatchConfig{
				MaxChunks:         100,
				MaxChunkSize:      10,
				MaxSubmissionTime: 1,
			},
			blockHeight:  10,
			latestHeight: 20,
			blockTime:    time.Unix(0, 1000*1000*1000+1).UTC(),
			expected:     false,
		},
		{
			name: "block time < last submission time + max submission time, block height == latest height",
			localBatchInfo: &executortypes.LocalBatchInfo{
				Start:              1,
				LastSubmissionTime: time.Unix(0, 0).UTC(),
			},
			batchConfig: executortypes.BatchConfig{
				MaxChunks:         100,
				MaxChunkSize:      10,
				MaxSubmissionTime: 1,
			},
			blockHeight:  10,
			latestHeight: 10,
			blockTime:    time.Unix(0, 1000*1000*1000).UTC(),
			expected:     true,
		},
		{
			name: "batch size >= (max chunks - 1) * max chunk size",
			localBatchInfo: &executortypes.LocalBatchInfo{
				Start:              1,
				LastSubmissionTime: time.Unix(0, 0).UTC(),
				BatchSize:          1000,
			},
			batchConfig: executortypes.BatchConfig{
				MaxChunks:         100,
				MaxChunkSize:      10,
				MaxSubmissionTime: 1,
			},
			blockHeight:  10,
			latestHeight: 20,
			blockTime:    time.Unix(0, 500).UTC(),
			expected:     true,
		},
		{
			name: "batch size < (max chunks - 1) * max chunk size",
			localBatchInfo: &executortypes.LocalBatchInfo{
				Start:              1,
				LastSubmissionTime: time.Unix(0, 0),
				BatchSize:          10,
			},
			batchConfig: executortypes.BatchConfig{
				MaxChunks:         100,
				MaxChunkSize:      10,
				MaxSubmissionTime: 1,
			},
			blockHeight:  10,
			latestHeight: 20,
			blockTime:    time.Unix(0, 500).UTC(),
			expected:     false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			batchSubmitter.localBatchInfo = tc.localBatchInfo
			batchSubmitter.batchCfg = tc.batchConfig
			actual := batchSubmitter.checkBatch(tc.blockHeight, tc.latestHeight, tc.blockTime)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestBatchFileSize(t *testing.T) {
	var err error

	batchSubmitter := BatchSubmitter{}
	batchSubmitter.batchFile, err = os.CreateTemp("", "batchfile")
	require.NoError(t, err)
	defer os.Remove(batchSubmitter.batchFile.Name())

	batchSubmitter.batchWriter, err = gzip.NewWriterLevel(batchSubmitter.batchFile, 6)
	require.NoError(t, err)
	defer batchSubmitter.batchWriter.Close()

	fileSize, err := batchSubmitter.batchFileSize(false)
	require.NoError(t, err)
	require.Equal(t, int64(0), fileSize)

	n, err := batchSubmitter.batchFile.Write([]byte("batch_bytes"))
	require.NoError(t, err)

	require.Equal(t, 11, n)
	fileSize, err = batchSubmitter.batchFileSize(false)
	require.NoError(t, err)
	require.Equal(t, int64(11), fileSize)
}

func TestEmptyBatchFile(t *testing.T) {
	var err error

	batchSubmitter := BatchSubmitter{}
	batchSubmitter.batchFile, err = os.CreateTemp("", "batchfile")
	require.NoError(t, err)
	defer os.Remove(batchSubmitter.batchFile.Name())

	_, err = batchSubmitter.batchFile.Write([]byte("batch_bytes"))
	require.NoError(t, err)

	fileSize, err := batchSubmitter.batchFileSize(false)
	require.NoError(t, err)
	require.Equal(t, int64(11), fileSize)

	err = batchSubmitter.emptyBatchFile()
	require.NoError(t, err)

	fileSize, err = batchSubmitter.batchFileSize(false)
	require.NoError(t, err)
	require.Equal(t, int64(0), fileSize)
}

func TestPrependLength(t *testing.T) {
	batchBytes := []byte("batch_bytes")
	lengthPrefixed := prependLength(batchBytes)

	require.Equal(t, append([]byte{0x0b, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, batchBytes...), lengthPrefixed)
}

func TestSubmitGenesis(t *testing.T) {
	baseDB, err := db.NewMemDB()
	require.NoError(t, err)

	batchDB := baseDB.WithPrefix([]byte("test_batch"))
	daDB := baseDB.WithPrefix([]byte("test_da"))

	appCodec, txConfig, err := childprovider.GetCodec("init")
	require.NoError(t, err)

	mockCaller := mockclient.NewMockCaller()
	testLogger, _ := zap.NewDevelopment()
	rpcClient, err := rpcclient.NewRPCClientWithClient(appCodec, client.NewWithCaller(mockCaller), []string{"http://localhost:26657"}, testLogger)
	require.NoError(t, err)
	batchNode := node.NewTestNode(nodetypes.NodeConfig{}, batchDB, appCodec, txConfig, rpcClient, nil)

	hostCdc, _, err := hostprovider.GetCodec("init")
	require.NoError(t, err)

	mockDA := NewMockDA(daDB, hostCdc, 1, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0")

	batchConfig := executortypes.BatchConfig{
		MaxChunks:    100,
		MaxChunkSize: 10,
	}

	batchSubmitter := BatchSubmitter{
		node:     batchNode,
		da:       mockDA,
		batchCfg: batchConfig,
		localBatchInfo: &executortypes.LocalBatchInfo{
			Start: 1,
			End:   0,
		},
		processedMsgs: make([]btypes.ProcessedMsgs, 0),
	}
	batchSubmitter.batchFile, err = os.CreateTemp("", "batchfile")
	require.NoError(t, err)
	defer os.Remove(batchSubmitter.batchFile.Name())

	batchSubmitter.batchWriter, err = gzip.NewWriterLevel(batchSubmitter.batchFile, 6)
	require.NoError(t, err)
	defer batchSubmitter.batchWriter.Close()

	mockCaller.SetRawCommit(10, []byte("commit_bytes"))
	genesisDoc := cmttypes.GenesisDoc{
		GenesisTime:   time.Unix(0, 1).UTC(),
		ChainID:       "test_chain",
		InitialHeight: 1,
	}
	mockCaller.SetGenesisDoc(genesisDoc)

	logger, observedLogs := logCapturer()
	ctx := types.NewContext(context.TODO(), logger, "")

	mockCount := int64(0)
	mockTimestampFetcher := func() int64 {
		mockCount++
		return mockCount
	}
	types.CurrentNanoTimestamp = mockTimestampFetcher

	err = batchSubmitter.submitGenesis(ctx)
	require.NoError(t, err)

	logs := observedLogs.TakeAll()
	require.Len(t, logs, 1)

	require.Equal(t, "submit genesis", logs[0].Message)
	require.Equal(t, []zapcore.Field{
		zap.Int("genesis size", 106),
		zap.Int("chunk length", 11),
	}, logs[0].Context)

	require.Len(t, batchSubmitter.processedMsgs, 11)

	require.Equal(t, btypes.ProcessedMsgs{
		Sender: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
		Msgs: []sdk.Msg{
			&ophosttypes.MsgRecordBatch{
				Submitter: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
				BridgeId:  1,
				BatchBytes: append([]byte{
					uint8(executortypes.BatchDataTypeGenesis),
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
				}, []byte("{\"genesis_")...),
			},
		},
		Timestamp: 1,
		Save:      true,
	}, batchSubmitter.processedMsgs[0])

	require.Equal(t, btypes.ProcessedMsgs{
		Sender: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
		Msgs: []sdk.Msg{
			&ophosttypes.MsgRecordBatch{
				Submitter: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
				BridgeId:  1,
				BatchBytes: append([]byte{
					uint8(executortypes.BatchDataTypeGenesis),
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
				}, []byte("time\":\"197")...),
			},
		},
		Timestamp: 2,
		Save:      true,
	}, batchSubmitter.processedMsgs[1])

	require.Equal(t, btypes.ProcessedMsgs{
		Sender: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
		Msgs: []sdk.Msg{
			&ophosttypes.MsgRecordBatch{
				Submitter: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
				BridgeId:  1,
				BatchBytes: append([]byte{
					uint8(executortypes.BatchDataTypeGenesis),
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
				}, []byte("0-01-01T00")...),
			},
		},
		Timestamp: 3,
		Save:      true,
	}, batchSubmitter.processedMsgs[2])

	require.Equal(t, btypes.ProcessedMsgs{
		Sender: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
		Msgs: []sdk.Msg{
			&ophosttypes.MsgRecordBatch{
				Submitter: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
				BridgeId:  1,
				BatchBytes: append([]byte{
					uint8(executortypes.BatchDataTypeGenesis),
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
				}, []byte(":00:00.000")...),
			},
		},
		Timestamp: 4,
		Save:      true,
	}, batchSubmitter.processedMsgs[3])

	require.Equal(t, btypes.ProcessedMsgs{
		Sender: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
		Msgs: []sdk.Msg{
			&ophosttypes.MsgRecordBatch{
				Submitter: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
				BridgeId:  1,
				BatchBytes: append([]byte{
					uint8(executortypes.BatchDataTypeGenesis),
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
				}, []byte("000001Z\",\"")...),
			},
		},
		Timestamp: 5,
		Save:      true,
	}, batchSubmitter.processedMsgs[4])

	require.Equal(t, btypes.ProcessedMsgs{
		Sender: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
		Msgs: []sdk.Msg{
			&ophosttypes.MsgRecordBatch{
				Submitter: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
				BridgeId:  1,
				BatchBytes: append([]byte{
					uint8(executortypes.BatchDataTypeGenesis),
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
				}, []byte("chain_id\":")...),
			},
		},
		Timestamp: 6,
		Save:      true,
	}, batchSubmitter.processedMsgs[5])

	require.Equal(t, btypes.ProcessedMsgs{
		Sender: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
		Msgs: []sdk.Msg{
			&ophosttypes.MsgRecordBatch{
				Submitter: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
				BridgeId:  1,
				BatchBytes: append([]byte{
					uint8(executortypes.BatchDataTypeGenesis),
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x06,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
				}, []byte("\"test_chai")...),
			},
		},
		Timestamp: 7,
		Save:      true,
	}, batchSubmitter.processedMsgs[6])

	require.Equal(t, btypes.ProcessedMsgs{
		Sender: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
		Msgs: []sdk.Msg{
			&ophosttypes.MsgRecordBatch{
				Submitter: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
				BridgeId:  1,
				BatchBytes: append([]byte{
					uint8(executortypes.BatchDataTypeGenesis),
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
				}, []byte("n\",\"initia")...),
			},
		},
		Timestamp: 8,
		Save:      true,
	}, batchSubmitter.processedMsgs[7])

	require.Equal(t, btypes.ProcessedMsgs{
		Sender: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
		Msgs: []sdk.Msg{
			&ophosttypes.MsgRecordBatch{
				Submitter: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
				BridgeId:  1,
				BatchBytes: append([]byte{
					uint8(executortypes.BatchDataTypeGenesis),
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
				}, []byte("l_height\":")...),
			},
		},
		Timestamp: 9,
		Save:      true,
	}, batchSubmitter.processedMsgs[8])

	require.Equal(t, btypes.ProcessedMsgs{
		Sender: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
		Msgs: []sdk.Msg{
			&ophosttypes.MsgRecordBatch{
				Submitter: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
				BridgeId:  1,
				BatchBytes: append([]byte{
					uint8(executortypes.BatchDataTypeGenesis),
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x09,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
				}, []byte("1,\"app_has")...),
			},
		},
		Timestamp: 10,
		Save:      true,
	}, batchSubmitter.processedMsgs[9])

	require.Equal(t, btypes.ProcessedMsgs{
		Sender: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
		Msgs: []sdk.Msg{
			&ophosttypes.MsgRecordBatch{
				Submitter: "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0",
				BridgeId:  1,
				BatchBytes: append([]byte{
					uint8(executortypes.BatchDataTypeGenesis),
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0a,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
				}, []byte("h\":\"\"}")...),
			},
		},
		Timestamp: 11,
		Save:      true,
	}, batchSubmitter.processedMsgs[10])
}
