package utils

func Map[T1, T2 any](slice []T1, fn func(T1) T2) []T2 {
	result := make([]T2, len(slice))
	for i, v := range slice {
		result[i] = fn(v)
	}
	return result
}

// MapErr applies fn to each element; stops and returns on the first error from fn.
func MapErr[T1, T2 any](slice []T1, fn func(T1) (T2, error)) ([]T2, error) {
	result := make([]T2, len(slice))
	for i, v := range slice {
		var err error
		result[i], err = fn(v)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// MapSkip applies fn to each element; skips the element if the error occurred.
func MapSkip[T1, T2 any](slice []T1, fn func(T1) (T2, error)) []T2 {
	result := make([]T2, 0, len(slice))
	for _, v := range slice {
		res, err := fn(v)
		if err != nil {
			continue
		}
		result = append(result, res)
	}
	return result
}

// MapUnique returns only unique values of the target field.
func MapUnique[T1 any, T2 comparable](slice []T1, fn func(T1) T2) []T2 {
	seen := make(map[T2]struct{}, len(slice))
	for _, v := range slice {
		seen[fn(v)] = struct{}{}
	}
	result := make([]T2, 0, len(seen))
	for v := range seen {
		result = append(result, v)
	}
	return result
}
