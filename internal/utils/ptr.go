package utils

func Ptr[T any](val T) *T {
	return &val
}

func Ref[T any](val T) *T {
	return Ptr(val)
}

func Unref[T any](val *T) T {
	if val == nil {
		var zero T
		return zero
	}
	return *val
}
