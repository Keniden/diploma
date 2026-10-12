package utils

import (
	"fmt"
	"math"
)

// LenToUint32Capped returns len(s) as uint32.
func LenToUint32Capped[T any](s []T) uint32 {
	v, err := IntToUint32(len(s))
	if err != nil {
		return math.MaxUint32
	}
	return v
}

// Int64ToInt32 returns v as int32 when it fits; otherwise an error.
func Int64ToInt32(v int64) (int32, error) {
	if v > math.MaxInt32 || v < math.MinInt32 {
		return 0, fmt.Errorf("value %d is out of int32 range", v)
	}

	return int32(v), nil
}

// Int64ToUint32 returns v as uint32 when it fits; otherwise an error.
func Int64ToUint32(v int64) (uint32, error) {
	if v < 0 || v > math.MaxUint32 {
		return 0, fmt.Errorf("value %d is out of uint32 range", v)
	}

	return uint32(v), nil
}

// Int64ToUint64 returns v as uint64 when it is non-negative; otherwise an error.
func Int64ToUint64(v int64) (uint64, error) {
	if v < 0 {
		return 0, fmt.Errorf("cannot convert negative value %d to uint64", v)
	}

	return uint64(v), nil
}

// IntToInt32 returns v as int32 when it fits; otherwise an error.
func IntToInt32(v int) (int32, error) {
	return Int64ToInt32(int64(v))
}

// IntToUint32 returns v as uint32 when it fits; otherwise an error.
func IntToUint32(v int) (uint32, error) {
	return Int64ToUint32(int64(v))
}
