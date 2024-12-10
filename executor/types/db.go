package types

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type WithdrawalData struct {
	Sequence       uint64 `json:"sequence"`
	From           string `json:"from"`
	To             string `json:"to"`
	Amount         uint64 `json:"amount"`
	BaseDenom      string `json:"base_denom"`
	WithdrawalHash []byte `json:"withdrawal_hash"`

	// extra info
	TxHeight int64  `json:"tx_height"`
	TxTime   int64  `json:"tx_time"`
	TxHash   string `json:"tx_hash"`
}

func NewWithdrawalData(
	sequence uint64,
	from string,
	to string,
	amount uint64,
	baseDenom string,
	withdrawalHash []byte,
	txHeight int64,
	txTime int64,
	txHash string,
) WithdrawalData {
	return WithdrawalData{
		Sequence:       sequence,
		From:           from,
		To:             to,
		Amount:         amount,
		BaseDenom:      baseDenom,
		WithdrawalHash: withdrawalHash,
		TxHeight:       txHeight,
		TxTime:         txTime,
		TxHash:         txHash,
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
	BlockTime   int64  `json:"block_time"`
	BlockHash   []byte `json:"block_hash"`
}

func NewTreeExtraData(
	blockNumber int64,
	blockTime int64,
	blockHash []byte,
) TreeExtraData {
	return TreeExtraData{
		BlockNumber: blockNumber,
		BlockTime:   blockTime,
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
