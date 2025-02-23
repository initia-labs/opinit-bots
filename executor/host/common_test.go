package host

import (
	"context"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

type mockChild struct {
	db             types.DB
	cdc            codec.Codec
	baseAccount    string
	oracleAccount  string
	nextL1Sequence uint64
	processedMsgs  []btypes.ProcessedMsgs
}

func NewMockChild(db types.DB, cdc codec.Codec, baseAccount string, oracleAccount string, nextL1Sequence uint64) *mockChild {
	return &mockChild{
		db:             db,
		cdc:            cdc,
		baseAccount:    baseAccount,
		oracleAccount:  oracleAccount,
		nextL1Sequence: nextL1Sequence,
		processedMsgs:  make([]btypes.ProcessedMsgs, 0),
	}
}

func (m *mockChild) DB() types.DB {
	return m.db
}

func (m *mockChild) Codec() codec.Codec {
	return m.cdc
}

func (m *mockChild) HasBroadcaster() bool {
	return m.baseAccount != "" || m.oracleAccount != ""
}

func (m *mockChild) BroadcastProcessedMsgs(msgs ...btypes.ProcessedMsgs) {
	m.processedMsgs = append(m.processedMsgs, msgs...)
}

func (m *mockChild) GetMsgFinalizeTokenDeposit(
	from string,
	to string,
	coin sdk.Coin,
	l1Sequence uint64,
	blockHeight int64,
	l1Denom string,
	data []byte,
) (sdk.Msg, string, error) {
	if m.baseAccount == "" {
		return nil, "", nil
	}
	return opchildtypes.NewMsgFinalizeTokenDeposit(
		m.baseAccount,
		from,
		to,
		coin,
		l1Sequence,
		types.MustInt64ToUint64(blockHeight),
		l1Denom,
		data,
	), m.baseAccount, nil
}

func (m *mockChild) GetMsgUpdateOracle(
	height int64,
	data []byte,
) (sdk.Msg, string, error) {
	if m.oracleAccount == "" {
		return nil, "", nil
	}
	msg := opchildtypes.NewMsgUpdateOracle(
		m.baseAccount,
		types.MustInt64ToUint64(height),
		data,
	)

	msgsAny := make([]*cdctypes.Any, 1)
	any, err := cdctypes.NewAnyWithValue(msg)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to create any")
	}
	msgsAny[0] = any

	return &authz.MsgExec{
		Grantee: m.oracleAccount,
		Msgs:    msgsAny,
	}, m.oracleAccount, nil
}

func (m *mockChild) GetMsgSetBridgeInfo(
	bridgeId uint64,
	bridgeConfig ophosttypes.BridgeConfig,
) (sdk.Msg, string, error) {
	if m.baseAccount == "" {
		return nil, "", nil
	}
	return opchildtypes.NewMsgSetBridgeInfo(
		m.baseAccount,
		opchildtypes.BridgeInfo{
			BridgeId:     bridgeId,
			BridgeConfig: bridgeConfig,
		},
	), m.baseAccount, nil
}

func (m *mockChild) QueryNextL1Sequence(ctx context.Context, height int64) (uint64, error) {
	if m.nextL1Sequence == 0 {
		return 0, errors.New("no next L1 sequence")
	}
	return m.nextL1Sequence, nil
}

var _ childNode = (*mockChild)(nil)

type mockBatchInfo struct {
	chain         string
	submitter     string
	outputIndex   uint64
	l2BlockNumber int64
}

type mockBatch struct {
	info *mockBatchInfo
}

func NewMockBatch() *mockBatch {
	return &mockBatch{}
}

func (m *mockBatch) UpdateBatchInfo(chain string, submitter string, outputIndex uint64, l2BlockNumber int64) {
	m.info = &mockBatchInfo{
		chain:         chain,
		submitter:     submitter,
		outputIndex:   outputIndex,
		l2BlockNumber: l2BlockNumber,
	}
}

var _ batchNode = (*mockBatch)(nil)

func logCapturer() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zap.DebugLevel)
	return zap.New(core), logs
}
