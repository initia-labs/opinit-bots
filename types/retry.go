package types

import (
	"context"
	"math"
	"math/rand/v2"
	"time"
)

// ~ 30 seconds
const MaxRetryCount = 4

// SleepWithRetry sleeps with exponential backoff.
func SleepWithRetry(ctx context.Context, retry int) bool {
	// to avoid to sleep too long
	if retry > MaxRetryCount {
		retry = MaxRetryCount
	} else if retry == 0 {
		return false
	}

	sleepTime := 2 * math.Exp2(float64(retry))
	sleepTime += rand.Float64() * float64(sleepTime) * 0.5 //nolint:all
	timer := time.NewTimer(time.Duration(sleepTime) * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return true
	case <-timer.C:
	}
	return false
}
