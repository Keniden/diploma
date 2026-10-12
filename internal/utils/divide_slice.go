package utils

import (
	"math"
	"slices"
)

// DivideSlice divides slice into smaller slices no more than sliceMaxSize
func DivideSlice[T any](slice []T, sliceMaxSize int) [][]T {
	if sliceMaxSize < 1 {
		return nil
	}

	divided := make([][]T, 0, int(math.Ceil(float64(len(slice))/float64(sliceMaxSize))))

	for chunk := range slices.Chunk(slice, sliceMaxSize) {
		divided = append(divided, chunk)
	}

	return divided
}
