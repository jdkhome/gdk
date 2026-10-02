package logs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestContextCorrelation(t *testing.T) {
	base := context.Background()
	identity := "private-agent@example.com"
	hash := sha256.Sum256([]byte(identity))
	ctx := WithRun(WithAgent(base, identity), "run\nforged=true")
	if AgentRef(ctx) != hex.EncodeToString(hash[:]) || AgentRef(base) != "" || AgentRef(WithAgent(base, "")) != "" {
		t.Fatal("agent关联错误")
	}
	if RunID(ctx) != "run\nforged=true" || RunID(base) != "" {
		t.Fatal("run关联错误")
	}
	out := &mockOutput{}
	logger := NewLogger(Level_Debug, []Output{out})
	logger.Info(ctx, nil, "message")
	logger.Info(base, nil, "plain")
	line := out.contents[0]
	if strings.Contains(line, identity) || strings.Contains(line, "\n") || !strings.Contains(line, "agent_ref="+AgentRef(ctx)) || !strings.Contains(line, `run_id="run\nforged=true"`) || !strings.Contains(line, "instance_id=") {
		t.Fatalf("关联字段不安全: %q", line)
	}
	if strings.Contains(out.contents[1], "agent_ref=") || strings.Contains(out.contents[1], "run_id=") {
		t.Fatal("context串线")
	}
	seen := sync.Map{}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := NewRunID()
			b, err := hex.DecodeString(id)
			if err != nil || len(b) < 16 {
				t.Error("随机ID格式错误")
			}
			if _, loaded := seen.LoadOrStore(id, true); loaded {
				t.Error("重复ID")
			}
		}()
	}
	wg.Wait()
}

type failingOutput struct{}

func (failingOutput) PushLog(context.Context, Level, string) error {
	return errors.New("SECRET-original-error")
}

func TestOutputFailureSafeAndContinues(t *testing.T) {
	outputFailure.Lock()
	outputFailure.last = time.Time{}
	outputFailure.Unlock()
	previous := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	defer func() { os.Stderr = previous; r.Close(); w.Close() }()
	out := &mockOutput{}
	logger := NewLogger(Level_Info, []Output{failingOutput{}, out, failingOutput{}})
	for i := 0; i < 20; i++ {
		logger.Error(context.Background(), nil, "SECRET-body")
	}
	w.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.contents) != 20 {
		t.Fatalf("失败后未继续输出: %d", len(out.contents))
	}
	if strings.Contains(string(b), "SECRET") || strings.Count(string(b), "\n") != 1 {
		t.Fatalf("stderr未安全限频: %q", b)
	}
}
