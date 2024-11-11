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
		Sender:           withdrawal.From,
		Sequence:         sequence,
		Amount:           amount.String(),
		Version:          []byte{ch.Version()},
		StorageRoot:      outputRoot,
		LatestBlockHash:  treeExtraData.BlockHash,
		BlockNumber:      treeExtraData.BlockNumber,
		Receiver:         withdrawal.To,
		WithdrawalHash:   withdrawal.WithdrawalHash,
	}, nil
}

func (ch Child) QueryWithdrawals(address string, offset uint64, limit uint64, descOrder bool) (res []executortypes.QueryWithdrawalResponse, err error) {
	withdrawals, err := ch.GetWithdrawalsByAddress(address, offset, limit, descOrder)
	if err != nil {
		return nil, err
	}

	for _, withdrawal := range withdrawals {
		proofs, outputIndex, outputRoot, extraDataBytes, err := ch.Merkle().GetProofs(withdrawal.Sequence)
		if err != nil {
			return nil, err
		}

		amount := sdk.NewCoin(withdrawal.BaseDenom, math.NewIntFromUint64(withdrawal.Amount))

		treeExtraData := executortypes.TreeExtraData{}
		err = json.Unmarshal(extraDataBytes, &treeExtraData)
		if err != nil {
			return nil, err
		}

		res = append(res, executortypes.QueryWithdrawalResponse{
			BridgeId:         ch.BridgeId(),
			OutputIndex:      outputIndex,
			WithdrawalProofs: proofs,
			Sender:           withdrawal.From,
			Sequence:         withdrawal.Sequence,
			Amount:           amount.String(),
			Version:          []byte{ch.Version()},
			StorageRoot:      outputRoot,
			LatestBlockHash:  treeExtraData.BlockHash,
			BlockNumber:      treeExtraData.BlockNumber,
			Receiver:         withdrawal.To,
			WithdrawalHash:   withdrawal.WithdrawalHash,
		})
	}
	return res, nil
}
