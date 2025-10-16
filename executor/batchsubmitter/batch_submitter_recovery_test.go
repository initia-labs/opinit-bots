package batchsubmitter

import (
	"compress/gzip"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/types"
)

func TestRecoverIncompleteBatchTruncatesCorruptedStream(t *testing.T) {
	t.Parallel()

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

	ctx := types.NewContext(context.TODO(), zap.NewNop(), "")
	recoveredBlocks, err := batchSubmitter.recoverIncompleteBatch(ctx)
	require.NoError(t, err)
	require.Len(t, recoveredBlocks, 1)
	require.Equal(t, block, recoveredBlocks[0])

	info, err = batchFile.Stat()
	require.NoError(t, err)
	require.Equal(t, int64(0), info.Size())
	require.Equal(t, int64(0), batchSubmitter.localBatchInfo.BatchSize)
}
