package logs

import (
	"context"
	"gopkg.in/natefinch/lumberjack.v2"
)

var _ Output = (*FileOutput)(nil)

type FileOutput struct {
	level  Level
	logger *lumberjack.Logger
}

func NewFileOutput(
	level Level,
	filePath string,
	maxSize, maxAge, maxBackups int,
) Output {
	return &FileOutput{
		level: level,
		logger: &lumberjack.Logger{
			Filename:   filePath,
			MaxSize:    maxSize,
			MaxAge:     maxAge,
			MaxBackups: maxBackups,
			LocalTime:  false, // 备份文件名是否使用本地时区
			Compress:   false, // 日志是否用gzip压缩
		},
	}
}

func (o *FileOutput) PushLog(ctx context.Context, level Level, content string) (err error) {
	if level.Value.level < o.level.Value.level {
		// 低级日志直接放弃
		return
	}

	_, err = o.logger.Write([]byte(content + "\n"))

	return
}
