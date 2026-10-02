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

func NewFileOutput(level Level, filePath string, maxSize, maxAge, maxBackups int) Output {
	return &FileOutput{level: level, logger: &lumberjack.Logger{
		Filename: filePath, MaxSize: maxSize, MaxAge: maxAge, MaxBackups: maxBackups,
		LocalTime: false, // 备份文件名使用UTC。
		Compress:  false,
	}}
}

func (o *FileOutput) PushLog(ctx context.Context, level Level, content string) (err error) {
	if level.Value.level < o.level.Value.level {
		return
	}
	_, err = o.logger.Write([]byte(content + "\n"))
	return
}

// Close 释放文件句柄；调用方需先停止向该输出写入。
func (o *FileOutput) Close() error { return o.logger.Close() }
