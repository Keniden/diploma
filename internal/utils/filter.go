package utils

func Filter[T any](slice []T, filter func(T) bool) []T {
	result := make([]T, 0)
	for _, v := range slice {
		if filter(v) {
			result = append(result, v)
		}
	}
	return result
}

func FilterOne[T any](slice []T, filter func(T) bool) (T, bool) {
	var res T
	for i := range slice {
		if filter(slice[i]) {
			return slice[i], true
		}
	}
	return res, false
}
