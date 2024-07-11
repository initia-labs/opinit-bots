package types

import (
	"encoding/binary"
	"fmt"
	"strconv"
)

const Splitter = '/'

func FromInt64(v int64) []byte {
	return []byte(fmt.Sprintf("%d", v))
}

func ToInt64(v []byte) int64 {
	data, err := strconv.ParseInt(string(v), 10, 64)
	if err != nil {
		// must not happen
		panic(err)
	}
	return data
}

func FromUInt64Key(v uint64) []byte {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, v)
	return data
}

func ToUInt64Key(data []byte) (v uint64) {
	return binary.BigEndian.Uint64(data)
}