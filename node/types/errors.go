package types

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

var ErrIgnoreAndTryLater = errors.New("try later")

func HandleErrIgnoreAndTryLater(ctx context.Context, err error) bool {
	if errors.Is(err, ErrIgnoreAndTryLater) {
		sleep := time.NewTimer(time.Minute)
		select {
		case <-ctx.Done():
		case <-sleep.C:
		}
		return true
	}
	return false
}
