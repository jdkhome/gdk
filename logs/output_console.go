package logs

import "context"

var _ Output = (*ConsoleOutput)(nil)

type ConsoleOutput struct {
	level Level
}

func NewConsoleOutput(
	level Level,
) Output {
	return &ConsoleOutput{
		level: level,
	}
}

func (o *ConsoleOutput) PushLog(ctx context.Context, level Level, content string) (err error) {
	if level.Value.level < o.level.Value.level {
		// 低级日志直接放弃
		return
	}
	println(content)
	return nil
}
