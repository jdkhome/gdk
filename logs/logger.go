package logs

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jdkhome/gdk/traces"
)

type Logger struct {
	Level   Level    // 接收日志的最低级别
	Outputs []Output // 日志输出
}

// 输出失败提示全进程限频，绝不包含原错误和日志正文。
var outputFailure struct {
	sync.Mutex
	last time.Time
}

func reportOutputFailure() {
	outputFailure.Lock()
	defer outputFailure.Unlock()
	now := time.Now()
	if !outputFailure.last.IsZero() && now.Sub(outputFailure.last) < time.Minute {
		return
	}
	outputFailure.last = now
	_, _ = fmt.Fprintln(os.Stderr, "logs: output failed; further failures suppressed for one minute")
}

func NewLogger(level Level, outputs []Output) *Logger {
	return &Logger{Level: level, Outputs: outputs}
}

func (l *Logger) Info(ctx context.Context, biz []string, msg string, args ...any) {
	l.push(ctx, Level_Info, biz, msg, args)
}
func (l *Logger) Debug(ctx context.Context, biz []string, msg string, args ...any) {
	l.push(ctx, Level_Debug, biz, msg, args)
}
func (l *Logger) Warn(ctx context.Context, biz []string, msg string, args ...any) {
	l.push(ctx, Level_Warn, biz, msg, args)
}
func (l *Logger) Error(ctx context.Context, biz []string, msg string, args ...any) {
	l.push(ctx, Level_Error, biz, msg, args)
}

func (l *Logger) push(ctx context.Context, level Level, biz []string, msg string, args []any) {
	if level.Value.level < l.Level.Value.level {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	tracer := traces.GetTracer(ctx)
	if tracer == nil {
		tracer = &traces.Tracer{}
	}
	content := fmt.Sprintf("%s [%s] [%s,%s] [%s] %s", time.Now().Format("2006-01-02 15:04:05.000"), level.Value.Name, tracer.GetTraceID(), tracer.GetSpanID(), strings.Join(biz, "|"), fmt.Sprintf(msg, args...))
	content += " instance_id=" + instanceID
	if ref := AgentRef(ctx); ref != "" {
		content += " agent_ref=" + ref
	}
	// 引号和控制字符转义，避免上下文中的运行ID注入日志字段。
	if run := RunID(ctx); run != "" {
		content += " run_id=" + strconv.Quote(run)
	}
	for _, output := range l.Outputs {
		var err error
		if category, ok := output.(CategoryOutput); ok {
			err = category.PushLogCategory(ctx, level, biz, content)
		} else {
			err = output.PushLog(ctx, level, content)
		}
		if err != nil {
			reportOutputFailure()
		}
	}
}
