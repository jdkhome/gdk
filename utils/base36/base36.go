package base36

import (
	"fmt"
	"math"
	"strings"
)

// 36进制字符集（0-9, A-Z）
const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func Uint64ToBase36(num uint64) string {
	if num == 0 {
		return "0"
	}

	result := make([]byte, 0, 20)
	for num > 0 {
		remainder := num % 36
		result = append(result, charset[remainder])
		num /= 36
	}

	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}

func Base36ToUint64(s string) (uint64, error) {
	s = strings.ToUpper(s)

	var result uint64

	for _, char := range s {
		idx := strings.IndexRune(charset, char)
		if idx == -1 {
			return 0, fmt.Errorf("invalid character '%c' in base36 string", char) // 新增错误检查
		}

		if result > (math.MaxUint64-uint64(idx))/36 {
			return 0, fmt.Errorf("base36 string '%s' overflows uint64", s)
		}

		result = result*36 + uint64(idx)
	}

	return result, nil
}
