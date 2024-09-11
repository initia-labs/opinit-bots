package host

import (
	"encoding/hex"
	"strconv"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
)

func ParseMsgRecordBatch(eventAttrs []abcitypes.EventAttribute) (
	submitter string, err error,
) {
	for _, attr := range eventAttrs {
		switch attr.Key {
		case ophosttypes.AttributeKeySubmitter:
			submitter = attr.Value
		}
	}
	return
}

func ParseMsgUpdateBatchInfo(eventAttrs []abcitypes.EventAttribute) (
	bridgeId uint64, submitter, chain string,
	outputIndex, l2BlockNumber uint64,
	err error) {
	for _, attr := range eventAttrs {
		switch attr.Key {
		case ophosttypes.AttributeKeyBridgeId:
			bridgeId, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return
			}
		case ophosttypes.AttributeKeyBatchChainType:
			chain = attr.Value
		case ophosttypes.AttributeKeyBatchSubmitter:
			submitter = attr.Value
		case ophosttypes.AttributeKeyFinalizedOutputIndex:
			outputIndex, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return
			}
		case ophosttypes.AttributeKeyFinalizedL2BlockNumber:
			l2BlockNumber, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return
			}
		}
	}
	return
}

func ParseMsgInitiateDeposit(eventAttrs []abcitypes.EventAttribute) (
	bridgeId, l1Sequence uint64,
	from, to, l1Denom, l2Denom, amount string,
	data []byte, err error) {

	for _, attr := range eventAttrs {
		switch attr.Key {
		case ophosttypes.AttributeKeyBridgeId:
			bridgeId, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return
			}
		case ophosttypes.AttributeKeyL1Sequence:
			l1Sequence, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return
			}
		case ophosttypes.AttributeKeyFrom:
			from = attr.Value
		case ophosttypes.AttributeKeyTo:
			to = attr.Value
		case ophosttypes.AttributeKeyL1Denom:
			l1Denom = attr.Value
		case ophosttypes.AttributeKeyL2Denom:
			l2Denom = attr.Value
		case ophosttypes.AttributeKeyAmount:
			amount = attr.Value
		case ophosttypes.AttributeKeyData:
			data, err = hex.DecodeString(attr.Value)
			if err != nil {
				return
			}
		}
	}
	return
}

func ParseMsgProposeOutput(eventAttrs []abcitypes.EventAttribute) (
	bridgeId, l2BlockNumber, outputIndex uint64,
	proposer string,
	outputRoot []byte,
	err error) {
	for _, attr := range eventAttrs {
		switch attr.Key {
		case ophosttypes.AttributeKeyProposer:
			proposer = attr.Value
		case ophosttypes.AttributeKeyBridgeId:
			bridgeId, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return
			}
		case ophosttypes.AttributeKeyOutputIndex:
			outputIndex, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return
			}
		case ophosttypes.AttributeKeyL2BlockNumber:
			l2BlockNumber, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return
			}
		case ophosttypes.AttributeKeyOutputRoot:
			outputRoot, err = hex.DecodeString(attr.Value)
			if err != nil {
				return
			}
		}
	}
	return
}

func ParseMsgFinalizeWithdrawal(eventAttrs []abcitypes.EventAttribute) (
	bridgeId, outputIndex, l2Sequence uint64,
	from, to, l1Denom, l2Denom, amount string,
	err error) {
	for _, attr := range eventAttrs {
		switch attr.Key {
		case ophosttypes.AttributeKeyBridgeId:
			bridgeId, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return
			}
		case ophosttypes.AttributeKeyOutputIndex:
			outputIndex, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return
			}
		case ophosttypes.AttributeKeyL2Sequence:
			l2Sequence, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return
			}
		case ophosttypes.AttributeKeyFrom:
			from = attr.Value
		case ophosttypes.AttributeKeyTo:
			to = attr.Value
		case ophosttypes.AttributeKeyL1Denom:
			l1Denom = attr.Value
		case ophosttypes.AttributeKeyL2Denom:
			l2Denom = attr.Value
		case ophosttypes.AttributeKeyAmount:
			amount = attr.Value
		}
	}
	return
}
