package types

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

var ErrIgnoreAndTryLater = errors.New("try later")

// HandleErrIgnoreAndTryLater handles the error and returns true if the error is ErrIgnoreAndTryLater.
// If the error is ErrIgnoreAndTryLater, it sleeps for a minute and returns true.
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
