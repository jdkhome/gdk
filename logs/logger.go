package logs

import (
	"context"
	"fmt"
	"github.com/jdkhome/gdk/traces"
	"strings"
	"time"
)

type Logger struct {
	Level   Level    // 接收日志的最低级别
	Outputs []Output // 日志输出
}

func NewLogger(level Level, outputs []Output) *Logger {
	return &Logger{
		Level:   level,
		Outputs: outputs,
	}
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
		// 低级日志直接放弃
		return
	}
	//var content string
	//if len(args) > 0 {
	//	content = fmt.Sprintf(msg, args...)
	//} else {
	//	content = msg
	//}

	tracer := traces.GetTracer(ctx)
	if tracer == nil {
		tracer = &traces.Tracer{}
	}

	content := fmt.Sprintf("%s [%s] [%s,%s] [%s] %s", time.Now().Format("2006-01-02 15:04:05.000"), level.Value.Name, tracer.GetTraceID(), tracer.GetSpanID(), strings.Join(biz, "|"), fmt.Sprintf(msg, args...))

	for _, output := range l.Outputs {
		if err := output.PushLog(ctx, level, content); err != nil {
			panic(err) // TODO 打日志报错直接蹦进程 不太好吧
		}
	}
}
