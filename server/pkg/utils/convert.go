package utils

import (
	"strconv"
	"strings"
)

func ConvertIntslice2String(arr []int64) string {
	strs := make([]string, len(arr))
	for idx, num := range arr {
		numStr := strconv.Itoa(int(num))
		strs[idx] = numStr
	}

	return strings.Join(strs, ",")
}