package child

import (
	"context"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type mockHost struct {
	db            types.DB
	cdc           codec.Codec
	bridgeId      uint64
	baseAccount   string
	outputs       map[uint64]ophosttypes.Output
	processedMsgs []btypes.ProcessedMsgs
}

func NewMockHost(db types.DB, cdc codec.Codec, bridgeId uint64, baseAccount string, outputs map[uint64]ophosttypes.Output) *mockHost {
	return &mockHost{
		db:            db,
		cdc:           cdc,
		bridgeId:      bridgeId,
		baseAccount:   baseAccount,
		outputs:       outputs,
		processedMsgs: make([]btypes.ProcessedMsgs, 0),
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
	if m.bridgeId != bridgeId {
		return nil, nil
	}

	lastIndex := uint64(0)
	for outputIndex := range m.outputs {
		if lastIndex < outputIndex {
			lastIndex = outputIndex
		}
	}

	if _, ok := m.outputs[lastIndex]; !ok {
		return nil, errors.New("collections: not found")
	}

	return &ophosttypes.QueryOutputProposalResponse{
		BridgeId:       bridgeId,
		OutputIndex:    lastIndex,
		OutputProposal: m.outputs[lastIndex],
	}, nil
}

func (m *mockHost) QueryOutput(ctx context.Context, bridgeId uint64, outputIndex uint64, height int64) (*ophosttypes.QueryOutputProposalResponse, error) {
	if m.bridgeId != bridgeId {
		return nil, nil
	}

	if _, ok := m.outputs[outputIndex]; !ok {
		return nil, errors.New("collections: not found")
	}

	return &ophosttypes.QueryOutputProposalResponse{
		BridgeId:       bridgeId,
		OutputIndex:    outputIndex,
		OutputProposal: m.outputs[outputIndex],
	}, nil
}

var _ hostNode = (*mockHost)(nil)

func logCapturer() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zap.DebugLevel)
	return zap.New(core), logs
}
