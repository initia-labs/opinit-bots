package child

import (
	"encoding/json"
	"errors"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	merkletypes "github.com/initia-labs/opinit-bots/merkle/types"
)

func (ch Child) QueryWithdrawal(sequence uint64) (executortypes.QueryWithdrawalResponse, error) {
	withdrawal, err := ch.GetWithdrawal(sequence)
	if err != nil {
		return executortypes.QueryWithdrawalResponse{}, err
	}

	amount := sdk.NewCoin(withdrawal.BaseDenom, math.NewIntFromUint64(withdrawal.Amount))

	res := executortypes.QueryWithdrawalResponse{
		BridgeId: ch.BridgeId(),
		From:     withdrawal.From,
		To:       withdrawal.To,
		Sequence: sequence,
		Amount:   amount,
		Version:  []byte{ch.Version()},
	}

	proofs, outputIndex, outputRoot, extraDataBytes, err := ch.Merkle().GetProofs(sequence)
	if errors.Is(err, merkletypes.ErrUnfinalizedTree) {
		// if the tree is not finalized, we just return only withdrawal info
		return res, nil
	} else if err != nil {
		return executortypes.QueryWithdrawalResponse{}, err
	}

	treeExtraData := executortypes.TreeExtraData{}
	err = json.Unmarshal(extraDataBytes, &treeExtraData)
	if err != nil {
		return executortypes.QueryWithdrawalResponse{}, err
	}
	res.WithdrawalProofs = proofs
	res.OutputIndex = outputIndex
	res.StorageRoot = outputRoot
	res.LastBlockHash = treeExtraData.BlockHash
	return res, nil
}

func (ch Child) QueryWithdrawals(address string, offset uint64, limit uint64, descOrder bool) (executortypes.QueryWithdrawalsResponse, error) {
	sequences, next, total, err := ch.GetSequencesByAddress(address, offset, limit, descOrder)
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
		Next:        next,
		Total:       total,
	}
	return res, nil
}
