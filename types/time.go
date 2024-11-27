package types

import "time"

var CurrentNanoTimestamp = func() int64 {
	return time.Now().UnixNano()
}
