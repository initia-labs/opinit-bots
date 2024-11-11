package child

import (
	"encoding/json"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	executortypes "github.com/initia-labs/opinit-bots/executor/types"
)

func (ch Child) QueryWithdrawal(sequence uint64) (executortypes.QueryWithdrawalResponse, error) {
	withdrawal, err := ch.GetWithdrawal(sequence)
	if err != nil {
		return executortypes.QueryWithdrawalResponse{}, err
	}

	proofs, outputIndex, outputRoot, extraDataBytes, err := ch.Merkle().GetProofs(sequence)
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
		From:             withdrawal.From,
		To:               withdrawal.To,
		Sequence:         sequence,
		Amount:           amount,
		Version:          []byte{ch.Version()},
		StorageRoot:      outputRoot,
		LastBlockHash:    treeExtraData.BlockHash,
		// BlockNumber:      treeExtraData.BlockNumber,
		// WithdrawalHash:   withdrawal.WithdrawalHash,
	}, nil
}

func (ch Child) QueryWithdrawals(address string, offset uint64, limit uint64, descOrder bool) (executortypes.QueryWithdrawalsResponse, error) {
	sequences, lastIndex, err := ch.GetSequencesByAddress(address, offset, limit, descOrder)
	if err != nil {
		return executortypes.QueryWithdrawalsResponse{}, err
	}
	withdrawals := make([]executortypes.QueryWithdrawalResponse, 0)
	for _, sequence := range sequences {
		withdrawal, err := ch.QueryWithdrawal(sequence)
		if err != nil {
			return executortypes.QueryWithdrawalsResponse{}, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}

	res := executortypes.QueryWithdrawalsResponse{
		Withdrawals: withdrawals,
		Total:       lastIndex,
	}
	return res, nil
}
