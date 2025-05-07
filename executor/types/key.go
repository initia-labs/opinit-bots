package types

import (
	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	"github.com/pkg/errors"
)

var (
	WithdrawalPrefix         = []byte("withdrawal")
	WithdrawalSequencePrefix = []byte("withdrawal_sequence")
	WithdrawalAddressPrefix  = []byte("withdrawal_address")

	WithdrawalSequenceKeyLength = len(WithdrawalSequencePrefix) + 1 + 8

	InternalStatusKey = []byte("internal_status")
)

func PrefixedWithdrawalSequence(sequence uint64) []byte {
	return dbtypes.GenerateKey([][]byte{
		WithdrawalSequencePrefix,
		dbtypes.FromUint64Key(sequence),
	})
}

func PrefixedWithdrawalAddress(address string) []byte {
	return dbtypes.GenerateKey([][]byte{
		WithdrawalAddressPrefix,
		[]byte(address),
	})
}

func PrefixedWithdrawalAddressSequence(address string, sequence uint64) []byte {
	return dbtypes.GenerateKey([][]byte{
		WithdrawalAddressPrefix,
		[]byte(address),
		dbtypes.FromUint64Key(sequence),
	})
}

func ParseWithdrawalSequenceKey(key []byte) (uint64, error) {
	if len(key) != WithdrawalSequenceKeyLength {
		return 0, errors.New("invalid key length")
	}
	return dbtypes.ToUint64Key(key[len(WithdrawalSequencePrefix)+1:]), nil
}

func ParseWithdrawalAddressSequenceKey(key []byte) (string, uint64, error) {
	if len(key) <= len(WithdrawalAddressPrefix)+1+8+1 {
		return "", 0, errors.New("invalid key length")
	}
	sequence := dbtypes.ToUint64Key(key[len(key)-8:])
	address := string(key[len(WithdrawalAddressPrefix)+1 : len(key)-8-1])
	return address, sequence, nil
}
