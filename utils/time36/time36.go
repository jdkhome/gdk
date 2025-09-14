package time36

import (
	"errors"
	"github.com/jdkhome/gdk/utils/base36"
	"strings"
	"time"
)

// 基准时间：2020-01-01 00:00:00 UTC
var baseTime = time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)

// TimeToTime36 将time.Time转换为time36字符串
// 规则：(输入时间 - 2020-01-01 00:00:00)的毫秒数 -> 36进制 -> 不足8位补0
func TimeToTime36(t time.Time) string {
	// 计算与基准时间的毫秒差
	duration := t.Sub(baseTime)
	milliseconds := uint64(duration.Milliseconds())

	// 转换为36进制
	base36Str := base36.Uint64ToBase36(milliseconds)

	// 不足8位补0
	if len(base36Str) < 8 {
		return padLeft(base36Str, 8, '0')
	}

	// 超过8位则截取后8位
	if len(base36Str) > 8 {
		return base36Str[len(base36Str)-8:]
	}

	return base36Str
}

// Time36ToTime 将time36字符串转换为time.Time
func Time36ToTime(s string) (time.Time, error) {
	// 验证输入长度
	if len(s) != 8 {
		return time.Time{}, errors.New("time36 string must be 8 characters long")
	}

	// 转换为毫秒数
	milliseconds, err := base36.Base36ToUint64(s)
	if err != nil {
		return time.Time{}, err
	}

	// 计算目标时间
	return baseTime.Add(time.Duration(milliseconds) * time.Millisecond), nil
}

// padLeft 在字符串左侧填充指定字符至指定长度
func padLeft(s string, length int, padChar rune) string {
	if len(s) >= length {
		return s
	}

	padCount := length - len(s)
	pad := strings.Repeat(string(padChar), padCount)
	return pad + s
}
