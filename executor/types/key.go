package types

import (
	dbtypes "github.com/initia-labs/opinit-bots-go/db/types"
)

var (
	WithdrawalsKey = []byte("withdrawals")
)

func PrefixedWithdrawalsKey(height uint64) []byte {
	return append(append(WithdrawalsKey, dbtypes.Splitter), dbtypes.FromUint64Key(height)...)
}
