package child

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"

	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/merkle"
)

func (ch Child) QueryWithdrawal(sequence uint64) (executortypes.QueryWithdrawalResponse, error) {
	withdrawal, err := ch.GetWithdrawal(sequence)
	if err != nil {
		return executortypes.QueryWithdrawalResponse{}, errors.Wrap(err, "failed to get withdrawal")
	}

	proofs, outputIndex, outputRoot, extraDataBytes, err := merkle.GetProofs(ch.DB(), sequence)
	if err != nil {
		return executortypes.QueryWithdrawalResponse{}, errors.Wrap(err, "failed to get proofs")
	}

	amount := sdk.NewCoin(withdrawal.BaseDenom, math.NewIntFromUint64(withdrawal.Amount))

	treeExtraData := executortypes.TreeExtraData{}
	err = treeExtraData.Unmarshal(extraDataBytes)
	if err != nil {
		return executortypes.QueryWithdrawalResponse{}, errors.Wrap(err, "failed to unmarshal tree extra data")
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
	}, nil
}

func (ch Child) QueryWithdrawals(address string, offset uint64, limit uint64, descOrder bool) (executortypes.QueryWithdrawalsResponse, error) {
	sequences, next, total, err := ch.GetSequencesByAddress(address, offset, limit, descOrder)
	if err != nil {
		return executortypes.QueryWithdrawalsResponse{}, errors.Wrap(err, "failed to get sequences by address")
	}
	withdrawals := make([]executortypes.QueryWithdrawalResponse, 0)
	for _, sequence := range sequences {
		withdrawal, err := ch.QueryWithdrawal(sequence)
		if err != nil {
			return executortypes.QueryWithdrawalsResponse{}, errors.Wrap(err, "failed to query withdrawal")
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
