package child

import (
	"testing"

	"github.com/initia-labs/opinit-bots/db"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/stretchr/testify/require"
)

func TestSaveGetWithdrawal(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	_, err = GetWithdrawal(db, 10)
	require.Error(t, err)

	_, err = GetWithdrawalByAddress(db, "to", 10)
	require.Error(t, err)

	withdrawal := executortypes.WithdrawalData{
		Sequence: 10,
		To:       "to",
	}
	err = SaveWithdrawal(db, withdrawal)
	require.NoError(t, err)

	w, err := GetWithdrawal(db, 10)
	require.NoError(t, err)
	require.Equal(t, withdrawal, w)

	sequence, err := GetWithdrawalByAddress(db, "to", 10)
	require.NoError(t, err)
	require.Equal(t, uint64(10), sequence)
}

func TestGetSequencesByAddress(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	for i := 1; i <= 50; i += 3 {
		withdrawal := executortypes.WithdrawalData{
			Sequence: uint64(i),
			To:       "addr0",
		}
		err = SaveWithdrawal(db, withdrawal)
		require.NoError(t, err)
	}

	cases := []struct {
		name      string
		address   string
		offset    uint64
		limit     uint64
		descOrder bool

		expectedSequences []uint64
		expectedNext      uint64
		expected          bool
	}{
		{
			name:      "offset 0, limit 10, desc",
			address:   "addr0",
			offset:    0,
			limit:     10,
			descOrder: true,

			expectedSequences: []uint64{49, 46, 43, 40, 37, 34, 31, 28, 25, 22},
			expectedNext:      19,
			expected:          false,
		},
		{
			name:      "offset 0, limit 10, asc",
			address:   "addr0",
			offset:    0,
			limit:     10,
			descOrder: false,

			expectedSequences: []uint64{1, 4, 7, 10, 13, 16, 19, 22, 25, 28},
			expectedNext:      31,
			expected:          false,
		},
		{
			name:      "offset 0, limit 100, asc",
			address:   "addr0",
			offset:    0,
			limit:     100,
			descOrder: false,

			expectedSequences: []uint64{1, 4, 7, 10, 13, 16, 19, 22, 25, 28, 31, 34, 37, 40, 43, 46, 49},
			expectedNext:      0,
			expected:          false,
		},
		{
			name:      "offset 26, limit 3, asc",
			address:   "addr0",
			offset:    26,
			limit:     3,
			descOrder: false,

			expectedSequences: []uint64{28, 31, 34},
			expectedNext:      37,
			expected:          false,
		},
		{
			name:      "offset 26, limit 3, desc",
			address:   "addr0",
			offset:    26,
			limit:     3,
			descOrder: true,

			expectedSequences: []uint64{25, 22, 19},
			expectedNext:      16,
			expected:          false,
		},
		{
			name:      "offset 100, limit 100, asc",
			address:   "addr0",
			offset:    100,
			limit:     100,
			descOrder: false,

			expectedSequences: nil,
			expectedNext:      0,
			expected:          false,
		},
		{
			name:      "addr1",
			address:   "addr1",
			offset:    0,
			limit:     10,
			descOrder: true,

			expectedSequences: nil,
			expectedNext:      0,
			expected:          false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sequences, next, err := GetSequencesByAddress(db, tc.address, tc.offset, tc.limit, tc.descOrder)

			if !tc.expected {
				require.NoError(t, err)
				require.Equal(t, tc.expectedSequences, sequences)
				require.Equal(t, tc.expectedNext, next)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestDeleteFutureWithdrawals(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	for i := 1; i <= 10; i++ {
		withdrawal := executortypes.WithdrawalData{
			Sequence: uint64(i),
			To:       "to",
		}
		err = SaveWithdrawal(db, withdrawal)
		require.NoError(t, err)
	}

	err = DeleteFutureWithdrawals(db, 11)
	require.NoError(t, err)
	for i := 1; i <= 10; i++ {
		_, err := GetWithdrawal(db, uint64(i))
		require.NoError(t, err)
		_, err = GetWithdrawalByAddress(db, "to", uint64(i))
		require.NoError(t, err)
	}

	err = DeleteFutureWithdrawals(db, 5)
	require.NoError(t, err)
	for i := 1; i <= 4; i++ {
		w, err := GetWithdrawal(db, uint64(i))
		require.NoError(t, err)
		require.Equal(t, uint64(i), w.Sequence)
		_, err = GetWithdrawalByAddress(db, "to", uint64(i))
		require.NoError(t, err)
	}
	for i := 5; i <= 10; i++ {
		_, err := GetWithdrawal(db, uint64(i))
		require.Error(t, err)
		_, err = GetWithdrawalByAddress(db, "to", uint64(i))
		require.Error(t, err)
	}

	err = DeleteFutureWithdrawals(db, 0)
	require.NoError(t, err)
	for i := 1; i <= 10; i++ {
		_, err := GetWithdrawal(db, uint64(i))
		require.Error(t, err)
		_, err = GetWithdrawalByAddress(db, "to", uint64(i))
		require.Error(t, err)
	}
}
