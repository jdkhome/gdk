package json_util

import "testing"

func TestToJsonStr(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"nil", nil, "null"},
		{"string", "hello", `"hello"`},
		{"int", 42, "42"},
		{"bool", true, "true"},
		{"slice", []int{1, 2, 3}, "[1,2,3]"},
		{"map", map[string]int{"a": 1}, `{"a":1}`},
		{"struct", struct {
			Name string `json:"name"`
		}{Name: "foo"}, `{"name":"foo"}`},
		{"html 转义禁用", "<a>", `"<a>"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToJsonStr(tt.in); got != tt.want {
				t.Errorf("ToJsonStr() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToJsonStrError(t *testing.T) {
	// 无法序列化的类型返回空字符串
	if got := ToJsonStr(make(chan int)); got != "" {
		t.Errorf("ToJsonStr(chan) = %q, want empty", got)
	}
	if got := ToJsonStr(func() {}); got != "" {
		t.Errorf("ToJsonStr(func) = %q, want empty", got)
	}
}
