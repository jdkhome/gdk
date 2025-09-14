package traces

import (
	"bytes"
	"context"
	"fmt"
	"github.com/duke-git/lancet/v2/random"
	"github.com/jdkhome/gdk/tx"
	"github.com/jdkhome/gdk/utils/ip_util"
	"github.com/jdkhome/gdk/utils/time36"
	"strconv"
	"strings"
	"time"
)

var (
	traceCounter = tx.NewGlobalCounter(0, 9999) // 默认范围从0到9999
	ipHash       = ip_util.GetIpHash()
	randomStr    = strings.ToUpper(random.RandNumeralOrLetter(6))
	ctxKey       = "CTX#TRACE"
)

type Tracer struct {
	traceID string
	spanID  []uint
}

func (t *Tracer) GetTraceID() string {
	return t.traceID
}

func (t *Tracer) GetSpanID() string {
	if len(t.spanID) == 0 {
		return ""
	}

	// 预估算缓冲区大小（每个uint最多10位数字，加上点分隔符）
	// 减少内存分配次数
	buf := bytes.NewBuffer(make([]byte, 0, len(t.spanID)*2))

	for i, num := range t.spanID {
		// 非第一个元素前加"."
		if i > 0 {
			buf.WriteByte('.')
		}
		// 直接写入数字的字符串形式
		buf.WriteString(strconv.FormatUint(uint64(num), 10))
	}

	return buf.String()
}

func NewTracer() *Tracer {
	var traceID strings.Builder
	traceID.Grow(20)
	traceID.WriteString(time36.TimeToTime36(time.Now()))          // 8位时间
	traceID.WriteString(ipHash)                                   // 6位IP哈希
	traceID.WriteString(randomStr)                                // 6位随机字符串
	traceID.WriteString(fmt.Sprintf("%04d", traceCounter.Next())) // 4位自增

	return &Tracer{
		traceID: traceID.String(),
		spanID:  []uint{0},
	}
}

func WithTracer(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKey, NewTracer())
}

func GetTracer(ctx context.Context) *Tracer {
	tracer, ok := ctx.Value(ctxKey).(*Tracer)
	if !ok {
		return nil
	}
	return tracer
}

func NewCtx() context.Context {
	return WithTracer(context.Background())
}
