package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrefixedWithdrawalSequence(t *testing.T) {
	bz := PrefixedWithdrawalSequence(0)
	require.Equal(t, bz, append([]byte("withdrawal_sequence/"), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}...))

	bz = PrefixedWithdrawalSequence(100)
	require.Equal(t, bz, append([]byte("withdrawal_sequence/"), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x64}...))
}

func TestPrefixedWithdrawalAddress(t *testing.T) {
	bz := PrefixedWithdrawalAddress("address")
	require.Equal(t, bz, []byte("withdrawal_address/address"))
}

func TestPrefixedWithdrawalAddressSequence(t *testing.T) {
	bz := PrefixedWithdrawalAddressSequence("address", 0)
	require.Equal(t, bz, append([]byte("withdrawal_address/address/"), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}...))

	bz = PrefixedWithdrawalAddressSequence("address", 100)
	require.Equal(t, bz, append([]byte("withdrawal_address/address/"), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x64}...))
}

func TestParseWithdrawalSequenceKey(t *testing.T) {
	_, err := ParseWithdrawalSequenceKey([]byte("withdrawal_sequence/"))
	require.Error(t, err)

	sequence, err := ParseWithdrawalSequenceKey(append([]byte("withdrawal_sequence/"), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x64}...))
	require.NoError(t, err)
	require.Equal(t, uint64(100), sequence)
}

func TestParseWithdrawalAddressSequenceKey(t *testing.T) {
	_, _, err := ParseWithdrawalAddressSequenceKey([]byte("withdrawal_address/address/"))
	require.Error(t, err)

	address, sequence, err := ParseWithdrawalAddressSequenceKey(append([]byte("withdrawal_address/address/"), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x64}...))
	require.NoError(t, err)
	require.Equal(t, "address", address)
	require.Equal(t, uint64(100), sequence)
}
