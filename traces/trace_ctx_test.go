package traces

import (
	"context"
	"testing"
)

func TestNewTracer(t *testing.T) {
	tr := NewTracer()
	if len(tr.GetTraceID()) != 24 {
		t.Errorf("TraceID 长度 = %d, want 24", len(tr.GetTraceID()))
	}
	if tr.GetSpanID() != "0" {
		t.Errorf("SpanID = %q, want 0", tr.GetSpanID())
	}
}

func TestGetSpanID(t *testing.T) {
	cases := []struct {
		name   string
		spanID []uint
		want   string
	}{
		{"nil", nil, ""},
		{"empty", []uint{}, ""},
		{"single", []uint{0}, "0"},
		{"multiple", []uint{1, 2, 3}, "1.2.3"},
		{"multi-digit", []uint{10, 200, 3000}, "10.200.3000"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr := &Tracer{spanID: tc.spanID}
			if got := tr.GetSpanID(); got != tc.want {
				t.Errorf("GetSpanID() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestWithTracerAndGetTracer(t *testing.T) {
	ctx := NewCtx()
	tr := GetTracer(ctx)
	if tr == nil {
		t.Fatal("GetTracer 返回 nil")
	}
	if len(tr.GetTraceID()) == 0 {
		t.Error("traceID 不应为空")
	}
}

func TestGetTracerNil(t *testing.T) {
	if GetTracer(context.Background()) != nil {
		t.Error("GetTracer 对空 context 应返回 nil")
	}
}
