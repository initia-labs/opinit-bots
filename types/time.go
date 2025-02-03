package types

import (
	"sync"
	"time"
)

var lastTimestamp int64
var tsMutex = sync.Mutex{}

// CurrentNanoTimestamp returns the current time in nanoseconds.
var CurrentNanoTimestamp = func() int64 {
	tsMutex.Lock()
	defer tsMutex.Unlock()
	res := time.Now().UTC().UnixNano()
	if res == lastTimestamp {
		res++
	}

	lastTimestamp = res
	return res
}
