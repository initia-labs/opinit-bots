package types

import "time"

// CurrentNanoTimestamp returns the current time in nanoseconds.
var CurrentNanoTimestamp = func() int64 {
	return time.Now().UnixNano()
}
