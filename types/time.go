package types

import "time"

var lastTimestamp int64

// CurrentNanoTimestamp returns the current time in nanoseconds.
var CurrentNanoTimestamp = func() int64 {
	res := time.Now().UTC().UnixNano()
	if res == lastTimestamp {
		res++
	}

	lastTimestamp = res
	return res
}
