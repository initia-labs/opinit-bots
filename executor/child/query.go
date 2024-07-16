package child

import (
	"encoding/json"
	"errors"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	"github.com/initia-labs/opinit-bots-go/node"
)

func (ch Child) GetAddressStr() (string, error) {
	addr := ch.node.GetAddress()
	if addr == nil {
		return "", errors.New("nil address")
	}
	return ch.ac.BytesToString(addr)
}

func (ch Child) QueryBridgeInfo() (opchildtypes.BridgeInfo, error) {
	req := &opchildtypes.QueryBridgeInfoRequest{}
	ctx := node.GetQueryContext(0)
	res, err := ch.opchildQueryClient.BridgeInfo(ctx, req)
	if err != nil {
		return opchildtypes.BridgeInfo{}, err
	}
	return res.BridgeInfo, nil
}

func (ch Child) QueryNextL1Sequence() (uint64, error) {
	req := &opchildtypes.QueryNextL1SequenceRequest{}
	ctx := node.GetQueryContext(0)
	res, err := ch.opchildQueryClient.NextL1Sequence(ctx, req)
	if err != nil {
		return 0, err
	}
	return res.NextL1Sequence, nil
}

func (ch Child) QueryNextL2Sequence(height uint64) (uint64, error) {
	req := &opchildtypes.QueryNextL2SequenceRequest{}
	ctx := node.GetQueryContext(height)
	res, err := ch.opchildQueryClient.NextL2Sequence(ctx, req)
	if err != nil {
		return 0, err
	}
	return res.NextL2Sequence, nil
}

func (ch Child) QueryWithdrawal(sequence uint64) (executortypes.QueryWithdrawalResponse, error) {
	withdrawalBytes, proofs, outputIndex, outputRoot, extraDataBytes, err := ch.mk.GetLeafWithProofs(sequence)
	if err != nil {
		return executortypes.QueryWithdrawalResponse{}, err
	}

	var withdrawal executortypes.WithdrawalData
	err = json.Unmarshal(withdrawalBytes, &withdrawal)
	if err != nil {
		return executortypes.QueryWithdrawalResponse{}, err
	}

	amount := sdk.NewCoin(withdrawal.BaseDenom, math.NewIntFromUint64(withdrawal.Amount))

	treeExtraData := executortypes.TreeExtraData{}
	err = json.Unmarshal(extraDataBytes, &treeExtraData)
	if err != nil {
		return executortypes.QueryWithdrawalResponse{}, err
	}

	return executortypes.QueryWithdrawalResponse{
		BridgeId:         ch.BridgeId(),
		OutputIndex:      outputIndex,
		WithdrawalProofs: proofs,
		Sender:           withdrawal.From,
		Sequence:         sequence,
		Amount:           amount.String(),
		Version:          []byte{ch.version},
		StorageRoot:      outputRoot,
		LatestBlockHash:  treeExtraData.BlockHash,

		BlockNumber:    treeExtraData.BlockNumber,
		Receiver:       withdrawal.To,
		WithdrawalHash: withdrawal.WithdrawalHash,
	}, nil
}
