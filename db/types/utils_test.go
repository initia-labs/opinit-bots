package types_test

import (
	"bytes"
	"testing"

	"github.com/initia-labs/opinit-bots-go/db/types"
	"github.com/stretchr/testify/require"
)

func TestUint64Key(t *testing.T) {
	bytes10 := types.FromUInt64Key(10)
	res10 := types.ToUInt64Key(bytes10)
	require.Equal(t, uint64(10), res10)

	bytes100 := types.FromUInt64Key(100)
	res100 := types.ToUInt64Key(bytes100)
	require.Equal(t, uint64(100), res100)

	require.True(t, bytes.Compare(bytes10, bytes100) < 0)
}
