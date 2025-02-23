package child

import (
	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
)

// GetWithdrawal returns the withdrawal data for the given sequence from the database
func GetWithdrawal(db types.BasicDB, sequence uint64) (executortypes.WithdrawalData, error) {
	dataBytes, err := db.Get(executortypes.PrefixedWithdrawalSequence(sequence))
	if err != nil {
		return executortypes.WithdrawalData{}, errors.Wrap(err, "failed to get withdrawal data from db")
	}
	data := executortypes.WithdrawalData{}
	err = data.Unmarshal(dataBytes)
	return data, err
}

func GetWithdrawalByAddress(db types.BasicDB, address string, sequence uint64) (uint64, error) {
	dataBytes, err := db.Get(executortypes.PrefixedWithdrawalAddressSequence(address, sequence))
	if err != nil {
		return 0, errors.Wrap(err, "failed to get withdrawal data sequence from db")
	}
	return dbtypes.ToUint64(dataBytes)
}

// GetSequencesByAddress returns the withdrawal sequences for the given address from the database
func GetSequencesByAddress(db types.DB, address string, offset uint64, limit uint64, descOrder bool) (sequences []uint64, next uint64, err error) {
	count := uint64(0)
	fetchFn := func(key, value []byte) (bool, error) {
		sequence, err := dbtypes.ToUint64(value)
		if err != nil {
			return true, errors.Wrap(err, "failed to convert value to uint64")
		}
		if count >= limit {
			next = sequence
			return true, nil
		}
		sequences = append(sequences, sequence)
		count++
		return false, nil
	}

	if descOrder {
		var startKey []byte
		if offset != 0 {
			startKey = executortypes.PrefixedWithdrawalAddressSequence(address, offset)
		}
		err = db.ReverseIterate(dbtypes.AppendSplitter(executortypes.PrefixedWithdrawalAddress(address)), startKey, fetchFn)
		if err != nil {
			return nil, 0, errors.Wrap(err, "failed to iterate withdrawal address indices")
		}
	} else {
		startKey := executortypes.PrefixedWithdrawalAddressSequence(address, offset)
		err := db.Iterate(dbtypes.AppendSplitter(executortypes.PrefixedWithdrawalAddress(address)), startKey, fetchFn)
		if err != nil {
			return nil, 0, err
		}
	}
	return sequences, next, nil
}

func SaveWithdrawal(db types.BasicDB, data executortypes.WithdrawalData) error {
	dataBytes, err := data.Marshal()
	if err != nil {
		return err
	}

	err = db.Set(executortypes.PrefixedWithdrawalSequence(data.Sequence), dataBytes)
	if err != nil {
		return errors.Wrap(err, "failed to save withdrawal data")
	}
	err = db.Set(executortypes.PrefixedWithdrawalAddressSequence(data.To, data.Sequence), dbtypes.FromUint64(data.Sequence))
	if err != nil {
		return errors.Wrap(err, "failed to save withdrawal address index")
	}
	return nil
}

// DeleteFutureWithdrawals deletes all future withdrawals from the database starting from the given sequence
func DeleteFutureWithdrawals(db types.DB, fromSequence uint64) error {
	var deletingKeys [][]byte

	err := db.Iterate(dbtypes.AppendSplitter(executortypes.WithdrawalSequencePrefix), executortypes.PrefixedWithdrawalSequence(fromSequence), func(key, value []byte) (bool, error) {
		data := executortypes.WithdrawalData{}
		err := data.Unmarshal(value)
		if err != nil {
			return true, err
		}
		deletingKeys = append(deletingKeys, executortypes.PrefixedWithdrawalAddressSequence(data.To, data.Sequence))
		deletingKeys = append(deletingKeys, key)
		return false, nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to delete future withdrawals")
	}
	for _, key := range deletingKeys {
		err = db.Delete(key)
		if err != nil {
			return errors.Wrap(err, "failed to delete withdrawal data")
		}
	}
	return nil
}
