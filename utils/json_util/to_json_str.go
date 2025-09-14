package json_util

import (
	"bytes"
	"encoding/json"
)

func ToJsonStr(o any) string {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false) // 禁用HTML转义
	if err := encoder.Encode(o); err != nil {
		return ""
	}
	// 移除Encode自动添加的换行符
	return buf.String()[:buf.Len()-1]
}
