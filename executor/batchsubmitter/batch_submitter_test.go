package batchsubmitter

import (
	"compress/gzip"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	executortypes "github.com/initia-labs/opinit-bots/executor/types"
)

func TestCheckBatchFileCorruption_Corrupted(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "batchfile")
	require.NoError(t, err)

	writer, err := gzip.NewWriterLevel(tmpFile, 6)
	require.NoError(t, err)

	block := []byte("block-bytes")
	_, err = writer.Write(prependLength(block))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.NoError(t, tmpFile.Close())

	// simulate crash by removing gzip trailer bytes
	reopen, err := os.OpenFile(tmpFile.Name(), os.O_RDWR, 0640)
	require.NoError(t, err)
	info, err := reopen.Stat()
	require.NoError(t, err)
	require.NoError(t, reopen.Truncate(info.Size()-8))
	require.NoError(t, reopen.Close())

	batchFile, err := os.OpenFile(tmpFile.Name(), os.O_RDWR, 0640)
	require.NoError(t, err)
	defer batchFile.Close()

	batchSubmitter := &BatchSubmitter{
		batchFile:      batchFile,
		localBatchInfo: &executortypes.LocalBatchInfo{Start: 1, End: 0},
	}

	corrupted, err := batchSubmitter.checkBatchFileCorruption()
	require.True(t, corrupted)
	require.NoError(t, err)
}

func TestCheckBatchFileCorruption_Success(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "batchfile")
	require.NoError(t, err)

	writer, err := gzip.NewWriterLevel(tmpFile, 6)
	require.NoError(t, err)

	block := []byte("block-bytes")
	_, err = writer.Write(prependLength(block))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.NoError(t, tmpFile.Close())

	batchFile, err := os.OpenFile(tmpFile.Name(), os.O_RDWR, 0640)
	require.NoError(t, err)

	writer, err = gzip.NewWriterLevel(batchFile, 6)
	require.NoError(t, err)

	_, err = writer.Write(prependLength(block))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.NoError(t, batchFile.Close())

	batchFile, err = os.OpenFile(tmpFile.Name(), os.O_RDWR, 0640)
	require.NoError(t, err)
	defer batchFile.Close()

	batchSubmitter := &BatchSubmitter{
		batchFile:      batchFile,
		localBatchInfo: &executortypes.LocalBatchInfo{Start: 1, End: 0},
	}

	corrupted, err := batchSubmitter.checkBatchFileCorruption()
	require.False(t, corrupted)
	require.NoError(t, err)
}
