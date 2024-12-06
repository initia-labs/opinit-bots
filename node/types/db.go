package types

import (
	dbtypes "github.com/initia-labs/opinit-bots/db/types"
)

func MarshalHeight(height int64) []byte {
	return dbtypes.FromInt64(height)
}

func UnmarshalHeight(heightBytes []byte) (int64, error) {
	return dbtypes.ToInt64(heightBytes)
}
