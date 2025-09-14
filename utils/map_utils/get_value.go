package map_utils

// GetValue 泛型方法：从 map[string]any 中获取指定类型的值
func GetValue[T any](m map[string]any, key string) (T, bool) {
	var zero T
	if m == nil {
		return zero, false
	}

	val, exists := m[key]
	if !exists {
		return zero, false
	}

	// 直接类型断言（需确保类型完全匹配）
	if v, ok := val.(T); ok {
		return v, true
	}

	return zero, false
}
