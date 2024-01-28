package log

import (
	"gdk/common"
	"testing"
)

func TestLoggerFactory(t *testing.T) {
	logger := LoggerFactory("/tmp/gdk/test.log", Text)
	logger.Info("测试日志打印", "key", "value")
	common.Cleanup()
}
