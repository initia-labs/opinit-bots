package types

import (
	"context"
	"math"
	"math/rand/v2"
	"time"
)

func SleepWithRetry(ctx context.Context, retry int) {
	// to avoid to sleep too long
	if retry > 7 {
		retry = 7
	} else if retry == 0 {
		return
	}

	sleepTime := 2 * math.Exp2(float64(retry))
	sleepTime += rand.Float64() * float64(sleepTime) * 0.5 //nolint:all
	timer := time.NewTimer(time.Duration(sleepTime) * time.Second)
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}
}
