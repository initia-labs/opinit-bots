package child

import (
	"strconv"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	"github.com/initia-labs/opinit-bots/merkle"
	"github.com/initia-labs/opinit-bots/node"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
)

func NewTestBaseChild(
	version uint8,

	node *node.Node,
	mk *merkle.Merkle,

	bridgeInfo ophosttypes.QueryBridgeResponse,

	initializeTreeFn func(int64) (bool, error),

	cfg nodetypes.NodeConfig,
) *BaseChild {
	return &BaseChild{
		version: version,

		node: node,
		mk:   mk,

		bridgeInfo: bridgeInfo,

		initializeTreeFn: initializeTreeFn,

		cfg: cfg,

		processedMsgs: make([]btypes.ProcessedMsgs, 0),
		msgQueue:      make(map[string][]sdk.Msg),
	}
}

func FinalizeDepositEvents(
	l1Sequence uint64,
	sender string,
	recipient string,
	denom string,
	baseDenom string,
	amount sdk.Coin,
	finalizeHeight uint64,
) []abcitypes.EventAttribute {
	return []abcitypes.EventAttribute{
		{
			Key:   opchildtypes.AttributeKeyL1Sequence,
			Value: strconv.FormatUint(l1Sequence, 10),
		},
		{
			Key:   opchildtypes.AttributeKeySender,
			Value: sender,
		},
		{
			Key:   opchildtypes.AttributeKeyRecipient,
			Value: recipient,
		},
		{
			Key:   opchildtypes.AttributeKeyDenom,
			Value: denom,
		},
		{
			Key:   opchildtypes.AttributeKeyBaseDenom,
			Value: baseDenom,
		},
		{
			Key:   opchildtypes.AttributeKeyAmount,
			Value: amount.Amount.String(),
		},
		{
			Key:   opchildtypes.AttributeKeyFinalizeHeight,
			Value: strconv.FormatUint(finalizeHeight, 10),
		},
	}
}

func UpdateOracleEvents(
	l1BlockHeight uint64,
	from string,
) []abcitypes.EventAttribute {
	return []abcitypes.EventAttribute{
		{
			Key:   opchildtypes.AttributeKeyHeight,
			Value: strconv.FormatUint(l1BlockHeight, 10),
		},
		{
			Key:   opchildtypes.AttributeKeyFrom,
			Value: from,
		},
	}
}

func InitiateWithdrawalEvents(
	from string,
	to string,
	denom string,
	baseDenom string,
	amount sdk.Coin,
	l2Sequence uint64,
) []abcitypes.EventAttribute {
	return []abcitypes.EventAttribute{
		{
			Key:   opchildtypes.AttributeKeyFrom,
			Value: from,
		},
		{
			Key:   opchildtypes.AttributeKeyTo,
			Value: to,
		},
		{
			Key:   opchildtypes.AttributeKeyDenom,
			Value: denom,
		},
		{
			Key:   opchildtypes.AttributeKeyBaseDenom,
			Value: baseDenom,
		},
		{
			Key:   opchildtypes.AttributeKeyAmount,
			Value: amount.Amount.String(),
		},
		{
			Key:   opchildtypes.AttributeKeyL2Sequence,
			Value: strconv.FormatUint(l2Sequence, 10),
		},
	}
}
