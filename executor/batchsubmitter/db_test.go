package batchsubmitter

import (
	"testing"
	"time"

	"github.com/initia-labs/opinit-bots/db"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/stretchr/testify/require"
)

func TestSaveGetLocalBatchInfo(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	info, err := GetLocalBatchInfo(db)
	require.NoError(t, err)
	require.Equal(t, executortypes.LocalBatchInfo{}, info)

	localBatchInfo := executortypes.LocalBatchInfo{
		Start:              1,
		End:                2,
		LastSubmissionTime: time.Unix(0, 10000),
		BatchSize:          1000,
	}

	err = SaveLocalBatchInfo(db, localBatchInfo)
	require.NoError(t, err)

	info, err = GetLocalBatchInfo(db)
	require.NoError(t, err)
	require.Equal(t, localBatchInfo, info)
}
