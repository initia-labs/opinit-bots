package host

import (
	"encoding/hex"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots/node"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
)

func NewTestBaseHost(version uint8, node *node.Node, bridgeInfo ophosttypes.QueryBridgeResponse, cfg nodetypes.NodeConfig, ophostQueryClient ophosttypes.QueryClient) *BaseHost {
	return &BaseHost{
		version:           version,
		node:              node,
		bridgeInfo:        bridgeInfo,
		cfg:               cfg,
		ophostQueryClient: ophostQueryClient,
		processedMsgs:     make([]btypes.ProcessedMsgs, 0),
		msgQueue:          make(map[string][]sdk.Msg),
	}
}

func RecordBatchEvents(
	submitter string,
) []abcitypes.EventAttribute {
	return []abcitypes.EventAttribute{
		{
			Key:   ophosttypes.AttributeKeySubmitter,
			Value: submitter,
		},
	}
}

func UpdateBatchInfoEvents(
	bridgeId uint64,
	chainType ophosttypes.BatchInfo_ChainType,
	submitter string,
	finalizedOutputIndex uint64,
	l2BlockNumber uint64,
) []abcitypes.EventAttribute {
	return []abcitypes.EventAttribute{
		{
			Key:   ophosttypes.AttributeKeyBridgeId,
			Value: strconv.FormatUint(bridgeId, 10),
		},
		{
			Key:   ophosttypes.AttributeKeyBatchChainType,
			Value: chainType.String(),
		},
		{
			Key:   ophosttypes.AttributeKeyBatchSubmitter,
			Value: submitter,
		},
		{
			Key:   ophosttypes.AttributeKeyFinalizedOutputIndex,
			Value: strconv.FormatUint(finalizedOutputIndex, 10),
		},
		{
			Key:   ophosttypes.AttributeKeyFinalizedL2BlockNumber,
			Value: strconv.FormatUint(l2BlockNumber, 10),
		},
	}
}

func InitiateTokenDepositEvents(
	bridgeId uint64,
	sender, to string,
	amount sdk.Coin,
	data []byte,
	l1Sequence uint64,
	l2Denom string,
) []abcitypes.EventAttribute {
	return []abcitypes.EventAttribute{
		{
			Key:   ophosttypes.AttributeKeyBridgeId,
			Value: strconv.FormatUint(bridgeId, 10),
		},
		{
			Key:   ophosttypes.AttributeKeyL1Sequence,
			Value: strconv.FormatUint(l1Sequence, 10),
		},
		{
			Key:   ophosttypes.AttributeKeyFrom,
			Value: sender,
		},
		{
			Key:   ophosttypes.AttributeKeyTo,
			Value: to,
		},
		{
			Key:   ophosttypes.AttributeKeyL1Denom,
			Value: amount.Denom,
		},
		{
			Key:   ophosttypes.AttributeKeyL2Denom,
			Value: l2Denom,
		},
		{
			Key:   ophosttypes.AttributeKeyAmount,
			Value: amount.Amount.String(),
		},
		{
			Key:   ophosttypes.AttributeKeyData,
			Value: hex.EncodeToString(data),
		},
	}
}

func ProposeOutputEvents(
	proposer string,
	bridgeId uint64,
	outputIndex uint64,
	l2BlockNumber int64,
	outputRoot []byte,
) []abcitypes.EventAttribute {
	return []abcitypes.EventAttribute{
		{
			Key:   ophosttypes.AttributeKeyProposer,
			Value: proposer,
		},
		{
			Key:   ophosttypes.AttributeKeyBridgeId,
			Value: strconv.FormatUint(bridgeId, 10),
		},
		{
			Key:   ophosttypes.AttributeKeyOutputIndex,
			Value: strconv.FormatUint(outputIndex, 10),
		},
		{
			Key:   ophosttypes.AttributeKeyL2BlockNumber,
			Value: strconv.FormatInt(l2BlockNumber, 10),
		},
		{
			Key:   ophosttypes.AttributeKeyOutputRoot,
			Value: hex.EncodeToString(outputRoot),
		},
	}
}

func FinalizeWithdrawalEvents(
	bridgeId uint64,
	outputIndex uint64,
	l2Sequence uint64,
	from string,
	to string,
	l1Denom string,
	l2Denom string,
	amount sdk.Coin,
) []abcitypes.EventAttribute {
	return []abcitypes.EventAttribute{
		{
			Key:   ophosttypes.AttributeKeyBridgeId,
			Value: strconv.FormatUint(bridgeId, 10),
		},
		{
			Key:   ophosttypes.AttributeKeyOutputIndex,
			Value: strconv.FormatUint(outputIndex, 10),
		},
		{
			Key:   ophosttypes.AttributeKeyL2Sequence,
			Value: strconv.FormatUint(l2Sequence, 10),
		},
		{
			Key:   ophosttypes.AttributeKeyFrom,
			Value: from,
		},
		{
			Key:   ophosttypes.AttributeKeyTo,
			Value: to,
		},
		{
			Key:   ophosttypes.AttributeKeyL1Denom,
			Value: l1Denom,
		},
		{
			Key:   ophosttypes.AttributeKeyL2Denom,
			Value: l2Denom,
		},
		{
			Key:   ophosttypes.AttributeKeyAmount,
			Value: amount.String(),
		},
	}
}

func UpdateOracleConfigEvents(
	bridgeId uint64,
	oracleEnabled bool,
) []abcitypes.EventAttribute {
	return []abcitypes.EventAttribute{
		{
			Key:   ophosttypes.AttributeKeyBridgeId,
			Value: strconv.FormatUint(bridgeId, 10),
		},
		{
			Key:   ophosttypes.AttributeKeyOracleEnabled,
			Value: strconv.FormatBool(oracleEnabled),
		},
	}
}
