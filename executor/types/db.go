package types

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type WithdrawalDataWithIndex struct {
	Withdrawal WithdrawalData `json:"withdrawal_data"`
	// index of the receiver address in db
	Index uint64 `json:"index"`
}

func NewWithdrawalDataWithIndex(
	withdrawal WithdrawalData,
	index uint64,
) WithdrawalDataWithIndex {
	return WithdrawalDataWithIndex{
		Withdrawal: withdrawal,
		Index:      index,
	}
}

func (w WithdrawalDataWithIndex) Marshal() ([]byte, error) {
	bz, err := json.Marshal(w)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal withdrawal data with index")
	}
	return bz, nil
}

func (w *WithdrawalDataWithIndex) Unmarshal(bz []byte) error {
	err := json.Unmarshal(bz, w)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal withdrawal data with index")
	}
	return nil
}

type WithdrawalData struct {
	Sequence       uint64 `json:"sequence"`
	From           string `json:"from"`
	To             string `json:"to"`
	Amount         uint64 `json:"amount"`
	BaseDenom      string `json:"base_denom"`
	WithdrawalHash []byte `json:"withdrawal_hash"`
}

func NewWithdrawalData(
	sequence uint64,
	from string,
	to string,
	amount uint64,
	baseDenom string,
	withdrawalHash []byte,
) WithdrawalData {
	return WithdrawalData{
		Sequence:       sequence,
		From:           from,
		To:             to,
		Amount:         amount,
		BaseDenom:      baseDenom,
		WithdrawalHash: withdrawalHash,
	}
}

func (w WithdrawalData) Marshal() ([]byte, error) {
	bz, err := json.Marshal(w)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal withdrawal data")
	}
	return bz, nil
}

func (w *WithdrawalData) Unmarshal(bz []byte) error {
	err := json.Unmarshal(bz, w)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal withdrawal data")
	}
	return nil
}

type TreeExtraData struct {
	BlockNumber int64  `json:"block_number"`
	BlockHash   []byte `json:"block_hash"`
}

func NewTreeExtraData(
	blockNumber int64,
	blockHash []byte,
) TreeExtraData {
	return TreeExtraData{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
	}
}

func (t TreeExtraData) Marshal() ([]byte, error) {
	bz, err := json.Marshal(t)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal tree extra data")
	}
	return bz, nil
}

func (t *TreeExtraData) Unmarshal(bz []byte) error {
	err := json.Unmarshal(bz, t)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal tree extra data")
	}
	return nil
}
