package child

import (
	"encoding/json"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	"github.com/initia-labs/opinit-bots-go/node"
)

func (ch Child) GetAddress() sdk.AccAddress {
	return ch.node.GetAddress()
}

func (ch Child) GetAddressStr() (string, error) {
	return ch.node.GetAddressString()
}

func (ch Child) QueryBridgeInfo() (opchildtypes.BridgeInfo, error) {
	req := &opchildtypes.QueryBridgeInfoRequest{}
	ctx, cancel := node.GetQueryContext(0)
	defer cancel()

	res, err := ch.opchildQueryClient.BridgeInfo(ctx, req)
	if err != nil {
		return opchildtypes.BridgeInfo{}, err
	}
	return res.BridgeInfo, nil
}

func (ch Child) QueryNextL1Sequence() (uint64, error) {
	req := &opchildtypes.QueryNextL1SequenceRequest{}
	ctx, cancel := node.GetQueryContext(0)
	defer cancel()

	res, err := ch.opchildQueryClient.NextL1Sequence(ctx, req)
	if err != nil {
		return 0, err
	}
	return res.NextL1Sequence, nil
}

func (ch Child) QueryNextL2Sequence(height uint64) (uint64, error) {
	req := &opchildtypes.QueryNextL2SequenceRequest{}
	ctx, cancel := node.GetQueryContext(height)
	defer cancel()

	res, err := ch.opchildQueryClient.NextL2Sequence(ctx, req)
	if err != nil {
		return 0, err
	}
	return res.NextL2Sequence, nil
}

func (ch Child) QueryWithdrawal(sequence uint64) (executortypes.QueryWithdrawalResponse, error) {
	withdrawal, err := ch.GetWithdrawal(sequence)
	if err != nil {
		return executortypes.QueryWithdrawalResponse{}, err
	}

	proofs, outputIndex, outputRoot, extraDataBytes, err := ch.mk.GetProofs(sequence)
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
		BlockNumber:      treeExtraData.BlockNumber,
		Receiver:         withdrawal.To,
		WithdrawalHash:   withdrawal.WithdrawalHash,
	}, nil
}
