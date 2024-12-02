package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLocalBatchInfo(t *testing.T) {
	batchInfo := LocalBatchInfo{
		Start:              1,
		End:                100,
		LastSubmissionTime: time.Unix(0, 10000).UTC(),
		BatchFileSize:      100,
	}

	bz, err := batchInfo.Marshal()
	require.NoError(t, err)

	batchInfo2 := LocalBatchInfo{}
	err = batchInfo2.Unmarshal(bz)
	require.NoError(t, err)

	require.Equal(t, batchInfo, batchInfo2)
}

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

func TestBatchDataChunk(t *testing.T) {
	start := uint64(1)
	end := uint64(100)
	index := uint64(0)
	length := uint64(100)
	chunkData := []byte("chunk")

	chunkDataData := MarshalBatchDataChunk(
		start,
		end,
		index,
		length,
		chunkData)
	require.Equal(t, 1+8+8+8+8+5, len(chunkDataData))

	chunk, err := UnmarshalBatchDataChunk(chunkDataData)
	require.NoError(t, err)

	require.Equal(t, start, chunk.Start)
	require.Equal(t, end, chunk.End)
	require.Equal(t, index, chunk.Index)
	require.Equal(t, length, chunk.Length)
	require.Equal(t, chunkData, chunk.ChunkData)
}
