package logs

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

type contextKey uint8

const (
	agentKey contextKey = iota
	runKey
)

var instanceID = NewRunID()

// WithAgent 只在上下文中保存完整身份的哈希，不保存原始身份。
func WithAgent(ctx context.Context, identity string) context.Context {
	if identity == "" {
		return ctx
	}
	sum := sha256.Sum256([]byte(identity))
	return context.WithValue(ctx, agentKey, hex.EncodeToString(sum[:]))
}

func AgentRef(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(agentKey).(string)
	return value
}

func WithRun(ctx context.Context, runID string) context.Context {
	return context.WithValue(ctx, runKey, runID)
}

func RunID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(runKey).(string)
	return value
}

// NewRunID 使用密码学随机数，避免暴露时间或进程信息。
func NewRunID() string {
	var id [16]byte
	_, err := rand.Read(id[:])
	if err != nil {
		panic("logs: secure random source unavailable")
	}
	return hex.EncodeToString(id[:])
}
