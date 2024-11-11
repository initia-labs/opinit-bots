package types

import (
	dbtypes "github.com/initia-labs/opinit-bots/db/types"
)

var (
	WithdrawalKey = []byte("withdrawal")
)

func PrefixedWithdrawalKey(sequence uint64) []byte {
	return append(append(WithdrawalKey, dbtypes.Splitter), dbtypes.FromUint64Key(sequence)...)
}

func PrefixedWithdrawalKeyAddress(address string) []byte {
	return append(append(WithdrawalKey, dbtypes.Splitter), []byte(address)...)
}

func PrefixedWithdrawalKeyAddressIndex(address string, index uint64) []byte {
	return append(append(PrefixedWithdrawalKeyAddress(address), dbtypes.Splitter), dbtypes.FromUint64Key(index)...)
}
