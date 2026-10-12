package utils

func GetHasNextAndTrim[T any](items []T, limit uint64) ([]T, bool) {
	if uint64(len(items)) > limit {
		return items[:limit], true
	}

	return items, false
}
