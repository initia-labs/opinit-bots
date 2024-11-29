package batchsubmitter

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type mockHost struct {
	batchInfos []ophosttypes.BatchInfoWithOutput
}

func NewMockHost(batchInfos []ophosttypes.BatchInfoWithOutput) *mockHost {
	return &mockHost{
		batchInfos: batchInfos,
	}
}

func (m *mockHost) QueryBatchInfos(ctx context.Context, bridgeId uint64) (*ophosttypes.QueryBatchInfosResponse, error) {
	return &ophosttypes.QueryBatchInfosResponse{
		BatchInfos: m.batchInfos,
	}, nil
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

var _ executortypes.DANode = (*mockDA)(nil)

func logCapturer() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zap.DebugLevel)
	return zap.New(core), logs
}
