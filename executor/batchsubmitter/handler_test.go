package batchsubmitter

import (
	"compress/gzip"
	"context"
	"os"
	"sync"
	"testing"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/gogoproto/proto"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
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
	"github.com/initia-labs/opinit-bots/txutils"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	cmttypes "github.com/cometbft/cometbft/types"

	"github.com/cosmos/cosmos-sdk/client/tx"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

func TestRawBlockHandler(t *testing.T) {
	baseDB, err := db.NewMemDB()
	require.NoError(t, err)

	batchDB := baseDB.WithPrefix([]byte("test_batch"))
	daDB := baseDB.WithPrefix([]byte("test_da"))

	appCodec, txConfig, err := childprovider.GetCodec("init")
	require.NoError(t, err)

	mockCaller := mockclient.NewMockCaller()
	rpcClient := rpcclient.NewRPCClientWithClient(appCodec, client.NewWithCaller(mockCaller))
	batchNode := node.NewTestNode(nodetypes.NodeConfig{}, batchDB, appCodec, txConfig, rpcClient, nil)

	hostCdc, _, err := hostprovider.GetCodec("init")
	require.NoError(t, err)

	mockDA := NewMockDA(daDB, hostCdc, 1, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0")

	batchConfig := executortypes.BatchConfig{
		MaxChunks:         100,
		MaxChunkSize:      10,
		MaxSubmissionTime: 1000,
	}

	batchSubmitter := BatchSubmitter{
		node:     batchNode,
		da:       mockDA,
		batchCfg: batchConfig,
		bridgeInfo: ophosttypes.QueryBridgeResponse{
			BridgeConfig: ophosttypes.BridgeConfig{
				SubmissionInterval: 150,
			},
		},
		batchInfoMu: &sync.Mutex{},
		batchInfos: []ophosttypes.BatchInfoWithOutput{
			{
				BatchInfo: ophosttypes.BatchInfo{
					ChainType: ophosttypes.BatchInfo_INITIA,
					Submitter: "submitter0",
				},
			},
		},
		processedMsgs: make([]btypes.ProcessedMsgs, 0),
		stage:         batchDB.NewStage(),
	}
	batchSubmitter.batchFile, err = os.CreateTemp("", "batchfile")
	require.NoError(t, err)
	defer os.Remove(batchSubmitter.batchFile.Name())

	batchSubmitter.batchWriter, err = gzip.NewWriterLevel(batchSubmitter.batchFile, 6)
	require.NoError(t, err)
	defer batchSubmitter.batchWriter.Close()

	createAuthzMsg := func(t *testing.T, sender string, msgs []sdk.Msg) *authz.MsgExec {
		msgsAny := make([]*cdctypes.Any, 0)
		for _, msg := range msgs {
			any, err := cdctypes.NewAnyWithValue(msg)
			require.NoError(t, err)
			msgsAny = append(msgsAny, any)
		}
		return &authz.MsgExec{
			Grantee: sender,
			Msgs:    msgsAny,
		}
	}

	mockCaller.SetRawCommit(1, []byte("commit_bytes"))
	genesisDoc := cmttypes.GenesisDoc{
		GenesisTime:   time.Unix(0, 1).UTC(),
		ChainID:       "test_chain",
		InitialHeight: 1,
	}

	mockCaller.SetGenesisDoc(genesisDoc)
	authzMsg := createAuthzMsg(t, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", []sdk.Msg{&opchildtypes.MsgUpdateOracle{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte("oracle_data2"), Height: 4}})
	txf := tx.Factory{}.WithChainID("test_chain").WithTxConfig(txConfig)
	pbb := &cmtproto.Block{
		Header: cmtproto.Header{
			Height: 1,
			Time:   time.Unix(0, 10).UTC(),
		},
		Data: cmtproto.Data{
			Txs: [][]byte{},
		},
	}

	txb, err := txf.BuildUnsignedTx(authzMsg)
	require.NoError(t, err)
	txBytes, err := txutils.EncodeTx(txConfig, txb.GetTx())
	require.NoError(t, err)
	pbb.Data.Txs = append(pbb.Data.Txs, txBytes)

	blockBytes, err := proto.Marshal(pbb)
	require.NoError(t, err)

	ctx := types.NewContext(context.TODO(), zap.NewNop(), "")

	err = SaveLocalBatchInfo(batchDB, executortypes.LocalBatchInfo{
		Start:              1,
		End:                0,
		LastSubmissionTime: time.Unix(0, 0).UTC(),
	})
	require.NoError(t, err)

	err = batchSubmitter.rawBlockHandler(ctx, nodetypes.RawBlockArgs{
		BlockHeight:  1,
		LatestHeight: 1,
		BlockBytes:   blockBytes,
	})
	require.NoError(t, err)
	// {"genesis_time":"1970-01-01T00:00:00.000000001Z","chain_id":"test_chain","initial_height":1,"app_hash":""}
	// length: 106
	require.Len(t, mockDA.processedMsgs, 11)

	require.Equal(
		t,
		mockDA.processedMsgs[0].Msgs[0].(*ophosttypes.MsgRecordBatch).BatchBytes,
		append([]byte{
			uint8(executortypes.BatchDataTypeGenesis),
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
		}, []byte("{\"genesis_")...),
	)
	require.Equal(
		t,
		mockDA.processedMsgs[1].Msgs[0].(*ophosttypes.MsgRecordBatch).BatchBytes,
		append([]byte{
			uint8(executortypes.BatchDataTypeGenesis),
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
		}, []byte("time\":\"197")...),
	)
	require.Equal(
		t,
		mockDA.processedMsgs[2].Msgs[0].(*ophosttypes.MsgRecordBatch).BatchBytes,
		append([]byte{
			uint8(executortypes.BatchDataTypeGenesis),
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
		}, []byte("0-01-01T00")...),
	)
	require.Equal(
		t,
		mockDA.processedMsgs[3].Msgs[0].(*ophosttypes.MsgRecordBatch).BatchBytes,
		append([]byte{
			uint8(executortypes.BatchDataTypeGenesis),
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
		}, []byte(":00:00.000")...),
	)
	require.Equal(
		t,
		mockDA.processedMsgs[4].Msgs[0].(*ophosttypes.MsgRecordBatch).BatchBytes,
		append([]byte{
			uint8(executortypes.BatchDataTypeGenesis),
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
		}, []byte("000001Z\",\"")...),
	)
	require.Equal(
		t,
		mockDA.processedMsgs[5].Msgs[0].(*ophosttypes.MsgRecordBatch).BatchBytes,
		append([]byte{
			uint8(executortypes.BatchDataTypeGenesis),
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
		}, []byte("chain_id\":")...),
	)
	require.Equal(
		t,
		mockDA.processedMsgs[6].Msgs[0].(*ophosttypes.MsgRecordBatch).BatchBytes,
		append([]byte{
			uint8(executortypes.BatchDataTypeGenesis),
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x06,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
		}, []byte("\"test_chai")...),
	)
	require.Equal(
		t,
		mockDA.processedMsgs[7].Msgs[0].(*ophosttypes.MsgRecordBatch).BatchBytes,
		append([]byte{
			uint8(executortypes.BatchDataTypeGenesis),
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
		}, []byte("n\",\"initia")...),
	)
	require.Equal(
		t,
		mockDA.processedMsgs[8].Msgs[0].(*ophosttypes.MsgRecordBatch).BatchBytes,
		append([]byte{
			uint8(executortypes.BatchDataTypeGenesis),
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
		}, []byte("l_height\":")...),
	)
	require.Equal(
		t,
		mockDA.processedMsgs[9].Msgs[0].(*ophosttypes.MsgRecordBatch).BatchBytes,
		append([]byte{
			uint8(executortypes.BatchDataTypeGenesis),
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x09,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
		}, []byte("1,\"app_has")...),
	)
	require.Equal(
		t,
		mockDA.processedMsgs[10].Msgs[0].(*ophosttypes.MsgRecordBatch).BatchBytes,
		append([]byte{
			uint8(executortypes.BatchDataTypeGenesis),
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0a,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
		}, []byte("h\":\"\"}")...),
	)

	syncedHeight, err := node.GetSyncInfo(batchDB)
	require.NoError(t, err)
	require.Equal(t, int64(1), syncedHeight)
	localBatchInfo, err := GetLocalBatchInfo(batchDB)
	require.NoError(t, err)
	require.Equal(t, executortypes.LocalBatchInfo{
		Start:              1,
		End:                0,
		LastSubmissionTime: time.Unix(0, 0).UTC(),
		BatchSize:          localBatchInfo.BatchSize,
	}, localBatchInfo)

	mockCaller.SetRawCommit(2, []byte("commit_bytes"))
	authzMsg = createAuthzMsg(t, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", []sdk.Msg{&opchildtypes.MsgUpdateOracle{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte("oracle_data2"), Height: 5}})
	pbb = &cmtproto.Block{
		Header: cmtproto.Header{
			Height: 2,
			Time:   time.Unix(0, 110).UTC(),
		},
		Data: cmtproto.Data{
			Txs: [][]byte{},
		},
	}

	txb, err = txf.BuildUnsignedTx(authzMsg)
	require.NoError(t, err)
	txBytes, err = txutils.EncodeTx(txConfig, txb.GetTx())
	require.NoError(t, err)
	pbb.Data.Txs = append(pbb.Data.Txs, txBytes)

	blockBytes, err = proto.Marshal(pbb)
	require.NoError(t, err)

	err = batchSubmitter.rawBlockHandler(ctx, nodetypes.RawBlockArgs{
		BlockHeight:  2,
		LatestHeight: 2,
		BlockBytes:   blockBytes,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(mockDA.processedMsgs), 1)

	syncedHeight, err = node.GetSyncInfo(batchDB)
	require.NoError(t, err)
	require.Equal(t, int64(2), syncedHeight)

	localBatchInfo, err = GetLocalBatchInfo(batchDB)
	require.NoError(t, err)
	require.Equal(t, executortypes.LocalBatchInfo{
		Start:              1,
		End:                2,
		LastSubmissionTime: time.Unix(0, 110).UTC(),
		BatchSize:          batchSubmitter.localBatchInfo.BatchSize,
	}, localBatchInfo)
}
