package types

import (
	dbtypes "github.com/initia-labs/opinit-bots/db/types"
)

var (
	WithdrawalPrefix         = []byte("withdrawal")
	WithdrawalSequencePrefix = []byte("withdrawal_sequence")
	WithdrawalAddressPrefix  = []byte("withdrawal_address")

	WithdrawalSequenceKeyLength = len(WithdrawalSequencePrefix) + 1 + 8
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

func PrefixedWithdrawalAddressIndex(address string, index uint64) []byte {
	return dbtypes.GenerateKey([][]byte{
		WithdrawalAddressPrefix,
		[]byte(address),
		dbtypes.FromUint64Key(index),
	})
}
