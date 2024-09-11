package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBatchDataHeader(t *testing.T) {
	start := uint64(1)
	end := uint64(100)

	chunks := [][]byte{
		[]byte("chunk1"),
		[]byte("chunk2"),
		[]byte("chunk3"),
	}

	checksums := make([][]byte, 0, len(chunks))
	for _, chunk := range chunks {
		checksum := GetChecksumFromChunk(chunk)
		checksums = append(checksums, checksum[:])
	}

	headerData := MarshalBatchDataHeader(
		start,
		end,
		checksums)
	require.Equal(t, 1+8+8+8+3*32, len(headerData))

	header, err := UnmarshalBatchDataHeader(headerData)
	require.NoError(t, err)

	require.Equal(t, start, header.Start)
	require.Equal(t, end, header.End)
	require.Equal(t, checksums, header.Checksums)
	require.Equal(t, len(chunks), len(header.Checksums))
}
