package logs

import (
	"context"
)

// CategoryOutput 在写入前额外接收业务分类，用于把特定分类路由到独立输出，
// 或将高噪声分类从控制台、汇总日志中剥离。普通输出不实现该接口，行为保持不变。
type CategoryOutput interface {
	Output
	PushLogCategory(ctx context.Context, level Level, biz []string, content string) error
}

// RoutedOutput 按业务分类首段过滤后转发给内层输出。
// include 非空时仅放行首段匹配的分类；exclude 非空时先跳过首段匹配的分类。
type RoutedOutput struct {
	inner   Output
	include []string
	exclude []string
}

// PushLog 无法获取分类信息，为保证高噪声分类不被误写入汇总日志，直接丢弃。
func (o *RoutedOutput) PushLog(_ context.Context, _ Level, _ string) error {
	return nil
}

// PushLogCategory 根据首段分类决定是否转发。
func (o *RoutedOutput) PushLogCategory(ctx context.Context, level Level, biz []string, content string) error {
	if matchAnyPrefix(o.exclude, biz) {
		return nil
	}
	if len(o.include) > 0 && !matchAnyPrefix(o.include, biz) {
		return nil
	}
	return o.inner.PushLog(ctx, level, content)
}

// ExcludeBiz 包装内层输出，跳过分类首段匹配的日志。
func ExcludeBiz(inner Output, prefixes ...string) Output {
	return &RoutedOutput{inner: inner, exclude: append([]string(nil), prefixes...)}
}

// IncludeBiz 包装内层输出，仅放行分类首段匹配的日志。
func IncludeBiz(inner Output, prefixes ...string) Output {
	return &RoutedOutput{inner: inner, include: append([]string(nil), prefixes...)}
}

// NewBizFileOutput 创建仅写入指定分类首段的滚动文件输出。
func NewBizFileOutput(level Level, prefixes []string, filePath string, maxSize, maxAge, maxBackups int) Output {
	return &RoutedOutput{
		inner:   NewFileOutput(level, filePath, maxSize, maxAge, maxBackups),
		include: append([]string(nil), prefixes...),
	}
}

// matchAnyPrefix 判断分类首段是否命中任意前缀；空分类视为不命中。
func matchAnyPrefix(prefixes []string, biz []string) bool {
	if len(biz) == 0 {
		return false
	}
	for _, p := range prefixes {
		if p != "" && biz[0] == p {
			return true
		}
	}
	return false
}
