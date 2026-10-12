package parser

import "strconv"

func toInt64(s string) (int64, bool) {
	val, err := strconv.ParseInt(s, 10, 64)
	return val, err == nil
}
