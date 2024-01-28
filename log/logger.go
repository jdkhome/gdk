package log

import (
	"gdk/common"
	"log/slog"
	"os"
	"path/filepath"
)

type HandlerType = int

const (
	Text HandlerType = iota
	Json
)

func LoggerFactory(path string, handler HandlerType) *slog.Logger {
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		panic(err)
	}

	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}

	// 注册清理
	common.RegisterCleanup(func() {
		_ = f.Close()
	})

	switch handler {
	case Text:
		return slog.New(slog.NewTextHandler(f, nil))
	case Json:
		return slog.New(slog.NewJSONHandler(f, nil))
	default:
		panic("未知handler")
	}

}
