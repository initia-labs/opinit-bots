package host

import (
	"encoding/hex"
	"fmt"
	"strconv"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
)

func missingAttrsError(missingAttrs map[string]struct{}) error {
	if len(missingAttrs) != 0 {
		missingAttrStr := ""
		for attr := range missingAttrs {
			missingAttrStr += attr + " "
		}
		return fmt.Errorf("missing attributes: %s", missingAttrStr)
	}
	return nil
}

func ParseMsgRecordBatch(eventAttrs []abcitypes.EventAttribute) (
	submitter string, err error,
) {
	missingAttrs := map[string]struct{}{
		ophosttypes.AttributeKeySubmitter: {},
	}

	for _, attr := range eventAttrs {
		switch attr.Key {
		case ophosttypes.AttributeKeySubmitter:
			submitter = attr.Value
		default:
			continue
		}
		delete(missingAttrs, attr.Key)
	}
	err = missingAttrsError(missingAttrs)
	return
}

func ParseMsgUpdateBatchInfo(eventAttrs []abcitypes.EventAttribute) (
	bridgeId uint64, submitter, chain string,
	outputIndex uint64,
	l2BlockNumber int64,
	err error) {
	missingAttrs := map[string]struct{}{
		ophosttypes.AttributeKeyBridgeId:               {},
		ophosttypes.AttributeKeyBatchChainType:         {},
		ophosttypes.AttributeKeyBatchSubmitter:         {},
		ophosttypes.AttributeKeyFinalizedOutputIndex:   {},
		ophosttypes.AttributeKeyFinalizedL2BlockNumber: {},
	}

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
			l2BlockNumber, err = strconv.ParseInt(attr.Value, 10, 64)
			if err != nil {
				return
			}
		default:
			continue
		}
		delete(missingAttrs, attr.Key)
	}
	err = missingAttrsError(missingAttrs)
	return
}

func ParseMsgInitiateDeposit(eventAttrs []abcitypes.EventAttribute) (
	bridgeId, l1Sequence uint64,
	from, to, l1Denom, l2Denom, amount string,
	data []byte, err error) {
	missingAttrs := map[string]struct{}{
		ophosttypes.AttributeKeyBridgeId:   {},
		ophosttypes.AttributeKeyL1Sequence: {},
		ophosttypes.AttributeKeyFrom:       {},
		ophosttypes.AttributeKeyTo:         {},
		ophosttypes.AttributeKeyL1Denom:    {},
		ophosttypes.AttributeKeyL2Denom:    {},
		ophosttypes.AttributeKeyAmount:     {},
		ophosttypes.AttributeKeyData:       {},
	}

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
		default:
			continue
		}
		delete(missingAttrs, attr.Key)
	}
	err = missingAttrsError(missingAttrs)
	return
}

func ParseMsgProposeOutput(eventAttrs []abcitypes.EventAttribute) (
	bridgeId uint64,
	l2BlockNumber int64,
	outputIndex uint64,
	proposer string,
	outputRoot []byte,
	err error) {
	missingAttrs := map[string]struct{}{
		ophosttypes.AttributeKeyProposer:      {},
		ophosttypes.AttributeKeyBridgeId:      {},
		ophosttypes.AttributeKeyOutputIndex:   {},
		ophosttypes.AttributeKeyL2BlockNumber: {},
		ophosttypes.AttributeKeyOutputRoot:    {},
	}

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
			l2BlockNumber, err = strconv.ParseInt(attr.Value, 10, 64)
			if err != nil {
				return
			}
		case ophosttypes.AttributeKeyOutputRoot:
			outputRoot, err = hex.DecodeString(attr.Value)
			if err != nil {
				return
			}
		default:
			continue
		}
		delete(missingAttrs, attr.Key)
	}
	err = missingAttrsError(missingAttrs)
	return
}

func ParseMsgFinalizeWithdrawal(eventAttrs []abcitypes.EventAttribute) (
	bridgeId, outputIndex, l2Sequence uint64,
	from, to, l1Denom, l2Denom, amount string,
	err error) {
	missingAttrs := map[string]struct{}{
		ophosttypes.AttributeKeyBridgeId:    {},
		ophosttypes.AttributeKeyOutputIndex: {},
		ophosttypes.AttributeKeyL2Sequence:  {},
		ophosttypes.AttributeKeyFrom:        {},
		ophosttypes.AttributeKeyTo:          {},
		ophosttypes.AttributeKeyL1Denom:     {},
		ophosttypes.AttributeKeyL2Denom:     {},
		ophosttypes.AttributeKeyAmount:      {},
	}

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
		default:
			continue
		}
		delete(missingAttrs, attr.Key)
	}
	err = missingAttrsError(missingAttrs)
	return
}
