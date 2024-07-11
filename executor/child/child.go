package child

import (
	"context"
	"time"

	"cosmossdk.io/core/address"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
	"go.uber.org/zap"

	"github.com/initia-labs/opinit-bots-go/node"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type hostNode interface {
	GetAddress() sdk.AccAddress
	BroadcastMsgs(nodetypes.ProcessedMsgs)
	RawKVProcessedData([]nodetypes.ProcessedMsgs, bool) ([]types.KV, error)
	QueryLastOutput() (ophosttypes.QueryOutputProposalResponse, error)
}

type Child struct {
	node *node.Node
	host hostNode

	bridgeId                  int64
	nextSentOutputTime        time.Time
	lastSentOutputBlockHeight uint64

	cfg    nodetypes.NodeConfig
	db     types.DB
	logger *zap.Logger

	cdc codec.Codec
	ac  address.Codec

	processedMsgs    []nodetypes.ProcessedMsgs
	msgQueue         []sdk.Msg
	blockWithdrawals executortypes.Withdrawals
}

func NewChild(bridgeId int64, cfg nodetypes.NodeConfig, db types.DB, logger *zap.Logger, cdc codec.Codec, txConfig client.TxConfig, host hostNode) *Child {
	node, err := node.NewNode(nodetypes.ChildNodeName, cfg, db, logger, cdc, txConfig)
	if err != nil {
		panic(err)
	}

	ch := &Child{
		node: node,
		host: host,

		bridgeId: bridgeId,

		cfg:    cfg,
		db:     db,
		logger: logger,

		cdc: cdc,
		ac:  cdc.InterfaceRegistry().SigningContext().AddressCodec(),

		processedMsgs: make([]nodetypes.ProcessedMsgs, 0),
		msgQueue:      make([]sdk.Msg, 0),
	}

	ch.registerHandlers()
	return ch
}

func (ch *Child) Start(ctx context.Context) {
	ch.node.Start(ctx)
}

func (ch *Child) registerHostNode(host hostNode) {
	ch.host = host
}

func (ch *Child) registerHandlers() {
	ch.node.RegisterBeginBlockHandler(ch.beginBlockHandler)
	ch.node.RegisterEventHandler(opchildtypes.EventTypeFinalizeTokenDeposit, ch.finalizeDepositHandler)
	ch.node.RegisterEventHandler(opchildtypes.EventTypeUpdateOracle, ch.updateOracleHandler)
	ch.node.RegisterEndBlockHandler(ch.endBlockHandler)
}

func (ch Child) GetAddress() sdk.AccAddress {
	return ch.node.GetAddress()
}

func (ch Child) BroadcastMsgs(msgs nodetypes.ProcessedMsgs) {
	ch.node.BroadcastMsgs(msgs)
}

func (ch Child) RawKVProcessedData(msgs []nodetypes.ProcessedMsgs, delete bool) ([]types.KV, error) {
	return ch.node.RawKVProcessedData(msgs, delete)
}

func (ch *Child) saveWithdrawals(withdrawals executortypes.Withdrawals) error {
	value, err := withdrawals.Marshal()
	if err != nil {
		return err
	}
	return ch.db.Set(executortypes.PrefixedWithdrawalsKey(withdrawals.Height), value)
}

func (ch *Child) loadWithdrawals(startHeight uint64, endHeight uint64) (res []executortypes.Withdrawals, err error) {
	iterErr := ch.db.Iterate(executortypes.PrefixedWithdrawalsKey(startHeight), executortypes.PrefixedWithdrawalsKey(endHeight+1), func(key, value []byte) bool {
		var withdrawals executortypes.Withdrawals
		err = withdrawals.Unmarshal(value)
		if err != nil {
			return true
		}
		res = append(res, withdrawals)
		return false
	})

	if iterErr != nil {
		return nil, iterErr
	}
	return nil, nil
}
