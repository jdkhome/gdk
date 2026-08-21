package map_utils

import "testing"

func TestGetValue(t *testing.T) {
	m := map[string]any{
		"str":  "hello",
		"int":  42,
		"bool": true,
		"nil":  nil,
	}

	if v, ok := GetValue[string](m, "str"); !ok || v != "hello" {
		t.Errorf("GetValue[str] = %v, %v", v, ok)
	}
	if v, ok := GetValue[int](m, "int"); !ok || v != 42 {
		t.Errorf("GetValue[int] = %v, %v", v, ok)
	}
	if v, ok := GetValue[bool](m, "bool"); !ok || v != true {
		t.Errorf("GetValue[bool] = %v, %v", v, ok)
	}
}

func TestGetValue_Miss(t *testing.T) {
	m := map[string]any{"int": 42}

	if _, ok := GetValue[int](m, "str"); ok {
		t.Error("类型不匹配应返回 ok=false")
	}
	if _, ok := GetValue[string](m, "missing"); ok {
		t.Error("key 不存在应返回 ok=false")
	}
	if _, ok := GetValue[string](nil, "str"); ok {
		t.Error("nil map 应返回 ok=false")
	}
	if _, ok := GetValue[string](map[string]any{"nil": nil}, "nil"); ok {
		t.Error("值为 nil 应返回 ok=false")
	}
}
