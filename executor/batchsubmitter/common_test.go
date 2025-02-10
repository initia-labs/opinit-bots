package batchsubmitter

import (
	"context"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type mockHost struct {
	batchInfos []ophosttypes.BatchInfoWithOutput
	blocks     map[int64]*cmttypes.Block
	chainId    string
}

func NewMockHost(batchInfos []ophosttypes.BatchInfoWithOutput, chainId string) *mockHost {
	return &mockHost{
		batchInfos: batchInfos,
		blocks:     make(map[int64]*cmttypes.Block),
		chainId:    chainId,
	}
}

func (m *mockHost) QueryBatchInfos(ctx types.Context, bridgeId uint64) ([]ophosttypes.BatchInfoWithOutput, error) {
	return m.batchInfos, nil
}

func (m *mockHost) QueryBlock(ctx context.Context, height int64) (*coretypes.ResultBlock, error) {
	if block, ok := m.blocks[height]; ok {
		return &coretypes.ResultBlock{
			Block: block,
		}, nil
	}
	return nil, errors.New("block not found")
}

func (m *mockHost) SetBlock(height int64, block *cmttypes.Block) {
	m.blocks[height] = block
}

func (m *mockHost) ChainId() string {
	return m.chainId
}

var _ hostNode = (*mockHost)(nil)

type mockDA struct {
	db            types.DB
	cdc           codec.Codec
	bridgeId      uint64
	baseAccount   string
	processedMsgs []btypes.ProcessedMsgs
}

func NewMockDA(db types.DB, cdc codec.Codec, bridgeId uint64, baseAccount string) *mockDA {
	return &mockDA{
		db:          db,
		cdc:         cdc,
		bridgeId:    bridgeId,
		baseAccount: baseAccount,
	}
}
func (m mockDA) Start(_ types.Context) {}

func (m *mockDA) DB() types.DB {
	return m.db
}

func (m *mockDA) Codec() codec.Codec {
	return m.cdc
}

func (m mockDA) HasBroadcaster() bool {
	return m.baseAccount != ""
}
func (m mockDA) CreateBatchMsg(batchBytes []byte) (sdk.Msg, string, error) {
	if m.baseAccount == "" {
		return nil, "", nil
	}

	msg := ophosttypes.NewMsgRecordBatch(
		m.baseAccount,
		m.bridgeId,
		batchBytes,
	)
	return msg, m.baseAccount, nil
}
func (m mockDA) GetNodeStatus() (nodetypes.Status, error) {
	return nodetypes.Status{}, nil
}

func (m *mockDA) BroadcastProcessedMsgs(msgs ...btypes.ProcessedMsgs) {
	m.processedMsgs = append(m.processedMsgs, msgs...)
}

func (m mockDA) LenProcessedBatchMsgs() (int, error) {
	return len(m.processedMsgs), nil
}

func (m mockDA) LenPendingBatchTxs() (int, error) {
	return 0, nil
}

var _ executortypes.DANode = (*mockDA)(nil)

func logCapturer() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zap.DebugLevel)
	return zap.New(core), logs
}
