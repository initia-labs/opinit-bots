package child

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type mockHost struct {
	db             types.DB
	cdc            codec.Codec
	baseAccount    string
	nextL1Sequence uint64
	nextL2Sequence uint64
	processedMsgs  []btypes.ProcessedMsgs
}

func NewMockHost(db types.DB, cdc codec.Codec, baseAccount string, nextL1Sequence uint64, nextL2Sequence uint64) *mockHost {
	return &mockHost{
		db:             db,
		cdc:            cdc,
		baseAccount:    baseAccount,
		nextL1Sequence: nextL1Sequence,
		nextL2Sequence: nextL2Sequence,
		processedMsgs:  make([]btypes.ProcessedMsgs, 0),
	}
}

func (m *mockHost) DB() types.DB {
	return m.db
}

func (m *mockHost) Codec() codec.Codec {
	return m.cdc
}

func (m *mockHost) HasBroadcaster() bool {
	return m.baseAccount != ""
}

func (m *mockHost) BroadcastProcessedMsgs(msgs ...btypes.ProcessedMsgs) {
	m.processedMsgs = append(m.processedMsgs, msgs...)
}

func (m *mockHost) GetMsgProposeOutput(
	bridgeId uint64,
	outputIndex uint64,
	l2BlockNumber int64,
	outputRoot []byte,
) (sdk.Msg, string, error) {
	if m.baseAccount == "" {
		return nil, "", nil
	}

	msg := ophosttypes.NewMsgProposeOutput(
		m.baseAccount,
		bridgeId,
		outputIndex,
		types.MustInt64ToUint64(l2BlockNumber),
		outputRoot,
	)
	return msg, m.baseAccount, nil
}

func (m *mockHost) QueryLastOutput(ctx context.Context, bridgeId uint64) (*ophosttypes.QueryOutputProposalResponse, error) {
	return nil, nil
}

func (m *mockHost) QueryOutput(ctx context.Context, bridgeId uint64, outputIndex uint64, height int64) (*ophosttypes.QueryOutputProposalResponse, error) {
	return nil, nil
}

var _ hostNode = (*mockHost)(nil)

func logCapturer() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zap.DebugLevel)
	return zap.New(core), logs
}
