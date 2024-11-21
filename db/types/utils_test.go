package types_test

import (
	"bytes"
	"testing"

	"github.com/initia-labs/opinit-bots/db/types"
	"github.com/stretchr/testify/require"
)

func TestFromToUint64Key(t *testing.T) {
	bytes0 := types.FromUint64Key(0)
	res0 := types.ToUint64Key(bytes0)
	require.Equal(t, uint64(0), res0)

	bytes10 := types.FromUint64Key(10)
	res10 := types.ToUint64Key(bytes10)
	require.Equal(t, uint64(10), res10)

	bytes100 := types.FromUint64Key(100)
	res100 := types.ToUint64Key(bytes100)
	require.Equal(t, uint64(100), res100)

	require.True(t, bytes.Compare(bytes10, bytes100) < 0)
}

func TestFromToInt64(t *testing.T) {
	bytesm10 := types.FromInt64(-10)
	require.Equal(t, []byte("-10"), bytesm10)
	resm10, err := types.ToInt64(bytesm10)
	require.NoError(t, err)
	require.Equal(t, int64(-10), resm10)

	bytes0 := types.FromInt64(0)
	require.Equal(t, []byte("0"), bytes0)
	res0, err := types.ToInt64(bytes0)
	require.NoError(t, err)
	require.Equal(t, int64(0), res0)

	bytes10 := types.FromInt64(10)
	require.Equal(t, []byte("10"), bytes10)
	res10, err := types.ToInt64(bytes10)
	require.NoError(t, err)
	require.Equal(t, int64(10), res10)
}

func TestFromToUInt64(t *testing.T) {
	bytes0 := types.FromUint64(0)
	require.Equal(t, []byte("0"), bytes0)
	res0, err := types.ToUint64(bytes0)
	require.NoError(t, err)
	require.Equal(t, uint64(0), res0)

	bytes10 := types.FromUint64(10)
	require.Equal(t, []byte("10"), bytes10)
	res10, err := types.ToUint64(bytes10)
	require.NoError(t, err)
	require.Equal(t, uint64(10), res10)
}

func TestGenerateKey(t *testing.T) {
	key := types.GenerateKey([][]byte{{1}, {2}, {3}})
	require.Equal(t, []byte{1, types.Splitter, 2, types.Splitter, 3}, key)
}

func TestAppendSplitter(t *testing.T) {
	key := types.AppendSplitter([]byte{1, 2, 3})
	require.Equal(t, []byte{1, 2, 3, types.Splitter}, key)
}
