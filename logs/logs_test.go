package logs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type mockOutput struct {
	mu       sync.Mutex
	levels   []Level
	contents []string
}

func (m *mockOutput) PushLog(ctx context.Context, level Level, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.levels = append(m.levels, level)
	m.contents = append(m.contents, content)
	return nil
}

func TestLevels(t *testing.T) {
	if len(Levels.Members()) != 4 {
		t.Errorf("Levels.Members() 长度 = %d, want 4", len(Levels.Members()))
	}

	if !Levels.CheckMember(Level_Info) {
		t.Error("Level_Info 应为成员")
	}

	if got, ok := Levels.GetByCode("info"); !ok || got != Level_Info {
		t.Errorf("GetByCode(info) = %v, %v", got, ok)
	}

	if _, ok := Levels.GetByCode("fatal"); ok {
		t.Error("GetByCode(fatal) 应返回 ok=false")
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	mock := &mockOutput{}
	logger := NewLogger(Level_Warn, []Output{mock})

	ctx := context.Background()
	logger.Debug(ctx, []string{"biz"}, "debug msg")
	logger.Info(ctx, []string{"biz"}, "info msg")
	logger.Warn(ctx, []string{"biz"}, "warn msg")
	logger.Error(ctx, []string{"biz"}, "error msg")

	if len(mock.levels) != 2 {
		t.Fatalf("应记录 2 条日志, 实际 %d", len(mock.levels))
	}
	if mock.levels[0] != Level_Warn {
		t.Errorf("第一条级别 = %v, want warn", mock.levels[0].Value.Name)
	}
	if mock.levels[1] != Level_Error {
		t.Errorf("第二条级别 = %v, want error", mock.levels[1].Value.Name)
	}
	if !strings.Contains(mock.contents[0], "warn msg") {
		t.Errorf("日志内容应包含 warn msg: %q", mock.contents[0])
	}
	if !strings.Contains(mock.contents[1], "error msg") {
		t.Errorf("日志内容应包含 error msg: %q", mock.contents[1])
	}
}

func TestFileOutput_PushLog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	out := NewFileOutput(Level_Info, path, 1, 1, 1)
	t.Cleanup(func() { _ = out.(*FileOutput).Close() })

	if err := out.PushLog(context.Background(), Level_Info, "hello"); err != nil {
		t.Fatalf("PushLog 错误: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取文件错误: %v", err)
	}
	if string(data) != "hello\n" {
		t.Errorf("文件内容 = %q, want %q", string(data), "hello\n")
	}
}

func TestFileOutput_LevelFilter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	out := NewFileOutput(Level_Warn, path, 1, 1, 1)

	if err := out.PushLog(context.Background(), Level_Info, "ignored"); err != nil {
		t.Fatalf("PushLog 错误: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("低级别日志不应创建文件, stat err = %v", err)
	}
}

func TestConsoleOutput(t *testing.T) {
	out := NewConsoleOutput(Level_Info)
	if err := out.PushLog(context.Background(), Level_Info, "hello"); err != nil {
		t.Errorf("PushLog 错误: %v", err)
	}
	if err := out.PushLog(context.Background(), Level_Debug, "ignored"); err != nil {
		t.Errorf("低级别 PushLog 应返回 nil: %v", err)
	}
}
