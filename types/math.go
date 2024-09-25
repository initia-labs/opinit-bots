package types

import (
	"errors"
	"math"
)

func SafeIntToUint8(v int) (uint8, error) {
	if v < 0 || v > math.MaxUint8 {
		return 0, errors.New("integer overflow conversion")
	}
	return uint8(v), nil
}

func SafeIntToUint32(v int) (uint32, error) {
	if v < 0 || v > math.MaxUint32 {
		return 0, errors.New("integer overflow conversion")
	}
	return uint32(v), nil
}

func SafeInt64ToUint64(v int64) (uint64, error) {
	if v < 0 {
		return 0, errors.New("integer overflow conversion")
	}
	return uint64(v), nil
}

func SafeUint64ToInt64(v uint64) (int64, error) {
	if v > math.MaxInt64 {
		return 0, errors.New("integer overflow conversion")
	}
	return int64(v), nil
}

func MustIntToUint8(v int) uint8 {
	ret, err := SafeIntToUint8(v)
	if err != nil {
		panic(err)
	}
	return ret
}

func MustIntToUint32(v int) uint32 {
	ret, err := SafeIntToUint32(v)
	if err != nil {
		panic(err)
	}
	return ret
}

func MustInt64ToUint64(v int64) uint64 {
	ret, err := SafeInt64ToUint64(v)
	if err != nil {
		panic(err)
	}
	return ret
}

func MustUint64ToInt64(v uint64) int64 {
	ret, err := SafeUint64ToInt64(v)
	if err != nil {
		panic(err)
	}
	return ret
}
