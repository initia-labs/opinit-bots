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

func GetSequencesByAddress(db types.DB, address string, offset uint64, limit uint64, descOrder bool) (sequences []uint64, next uint64, err error) {
	if limit == 0 {
		return nil, 0, nil
	}

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

func SaveWithdrawal(db types.BasicDB, sequence uint64, data executortypes.WithdrawalData) error {
	dataBytes, err := data.Marshal()
	if err != nil {
		return err
	}

	err = db.Set(executortypes.PrefixedWithdrawalSequence(sequence), dataBytes)
	if err != nil {
		return errors.Wrap(err, "failed to save withdrawal data")
	}
	err = db.Set(executortypes.PrefixedWithdrawalAddressSequence(data.To, sequence), dbtypes.FromUint64(sequence))
	if err != nil {
		return errors.Wrap(err, "failed to save withdrawal address index")
	}
	return nil
}

func DeleteFutureWithdrawals(db types.DB, fromSequence uint64) error {
	return db.Iterate(dbtypes.AppendSplitter(executortypes.WithdrawalSequencePrefix), nil, func(key, value []byte) (bool, error) {
		sequence := dbtypes.ToUint64Key(key[len(key)-8:])
		if sequence < fromSequence {
			return false, nil
		}

		data := executortypes.WithdrawalData{}
		err := data.Unmarshal(value)
		if err != nil {
			return true, err
		}
		err = db.Delete(executortypes.PrefixedWithdrawalAddressSequence(data.To, data.Sequence))
		if err != nil {
			return true, errors.Wrap(err, "failed to delete withdrawal address index")
		}
		err = db.Delete(key)
		if err != nil {
			return true, errors.Wrap(err, "failed to delete withdrawal data")
		}

		return false, nil
	})
}
