package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithdrawalData(t *testing.T) {
	wd := WithdrawalData{
		Sequence:       100,
		From:           "from",
		To:             "to",
		Amount:         100,
		BaseDenom:      "base_denom",
		WithdrawalHash: []byte("withdrawal_hash"),
	}
	wd2 := NewWithdrawalData(100, "from", "to", 100, "base_denom", []byte("withdrawal_hash"))
	require.Equal(t, wd, wd2)

	bz, err := wd.Marshal()
	require.NoError(t, err)

	wd3 := WithdrawalData{}
	err = wd3.Unmarshal(bz)
	require.NoError(t, err)
	require.Equal(t, wd, wd3)
}

func TestTreeExtraData(t *testing.T) {
	td := TreeExtraData{
		BlockNumber: 100,
		BlockHash:   []byte("block_hash"),
	}
	td2 := NewTreeExtraData(100, []byte("block_hash"))
	require.Equal(t, td, td2)

	bz, err := td.Marshal()
	require.NoError(t, err)

	td3 := TreeExtraData{}
	err = td3.Unmarshal(bz)
	require.NoError(t, err)
	require.Equal(t, td, td3)
}
