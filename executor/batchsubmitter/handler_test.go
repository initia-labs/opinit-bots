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
					ChainType: ophosttypes.BatchInfo_CHAIN_TYPE_INITIA,
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
	require.Len(t, mockDA.processedMsgs, 0)

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
