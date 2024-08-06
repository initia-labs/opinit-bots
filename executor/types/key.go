package types

import (
	dbtypes "github.com/initia-labs/opinit-bots-go/db/types"
)

var (
	WithdrawalKey = []byte("withdrawal")
)

func PrefixedWithdrawalKey(sequence uint64) []byte {
	return append(append(WithdrawalKey, dbtypes.Splitter), dbtypes.FromUint64Key(sequence)...)
}
