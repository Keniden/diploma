package utils

// Uniq returns a slice with duplicates removed, keeping first occurrences.
func Uniq[T comparable](in []T) []T {
	if len(in) == 0 {
		return in
	}
	seen := make(map[T]struct{}, len(in))
	out := make([]T, 0, len(in))

	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
