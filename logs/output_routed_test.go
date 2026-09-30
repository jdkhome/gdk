package logs

import (
	"context"
	"testing"
)

// routedCapture 记录转发到内层的日志内容，用于验证分类路由。
type routedCapture struct {
	lines []string
}

func (o *routedCapture) PushLog(_ context.Context, _ Level, content string) error {
	o.lines = append(o.lines, content)
	return nil
}

func TestRoutedOutputExclude(t *testing.T) {
	inner := &routedCapture{}
	out := ExcludeBiz(inner, "db_sql", "http_request").(*RoutedOutput)

	// 匹配排除前缀的分类被丢弃。
	_ = out.PushLogCategory(context.Background(), Level_Info, []string{"db_sql"}, "sql")
	_ = out.PushLogCategory(context.Background(), Level_Info, []string{"http_request"}, "http")
	// 非匹配分类被转发。
	_ = out.PushLogCategory(context.Background(), Level_Info, []string{"chat"}, "chat")

	if len(inner.lines) != 1 || inner.lines[0] != "chat" {
		t.Fatalf("排除路由错误，得到 %#v", inner.lines)
	}
}

func TestRoutedOutputInclude(t *testing.T) {
	inner := &routedCapture{}
	out := IncludeBiz(inner, "db_sql").(*RoutedOutput)

	_ = out.PushLogCategory(context.Background(), Level_Info, []string{"db_sql"}, "sql")
	_ = out.PushLogCategory(context.Background(), Level_Info, []string{"chat"}, "chat")

	if len(inner.lines) != 1 || inner.lines[0] != "sql" {
		t.Fatalf("包含路由错误，得到 %#v", inner.lines)
	}
}

func TestRoutedOutputWithoutCategoryDrops(t *testing.T) {
	inner := &routedCapture{}
	out := IncludeBiz(inner, "db_sql").(*RoutedOutput)

	// 无分类信息时不写入，避免高噪声日志误入汇总。
	if err := out.PushLog(context.Background(), Level_Info, "sql"); err != nil {
		t.Fatalf("PushLog 返回错误: %v", err)
	}
	if len(inner.lines) != 0 {
		t.Fatalf("无分类时不应写入，得到 %#v", inner.lines)
	}
}

func TestMatchAnyPrefix(t *testing.T) {
	cases := []struct {
		prefixes []string
		biz      []string
		want     bool
	}{
		{[]string{"db_sql"}, []string{"db_sql"}, true},
		{[]string{"db_sql"}, []string{"db_sql", "query"}, true},
		{[]string{"db_sql"}, []string{"chat"}, false},
		{[]string{"db_sql", "http"}, []string{"http"}, true},
		{[]string{""}, []string{"db_sql"}, false},
		{nil, []string{"db_sql"}, false},
		{[]string{"db_sql"}, nil, false},
	}
	for _, c := range cases {
		if got := matchAnyPrefix(c.prefixes, c.biz); got != c.want {
			t.Errorf("matchAnyPrefix(%v, %v) = %v, 期望 %v", c.prefixes, c.biz, got, c.want)
		}
	}
}
