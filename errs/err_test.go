package errs

import (
	"errors"
	"fmt"
	"testing"
)

// 测试Err的Error()方法
func TestErr_Error(t *testing.T) {
	tests := []struct {
		name string
		e    *Err
		want string
	}{
		{
			name: "正常错误信息",
			e:    &Err{Code: "test_code", Msg: "test message"},
			want: "[test_code]test message",
		},
		{
			name: "空消息",
			e:    &Err{Code: "empty_msg", Msg: ""},
			want: "[empty_msg]",
		},
		{
			name: "空代码",
			e:    &Err{Code: "", Msg: "no code"},
			want: "[]no code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

// 测试Is函数
func TestIs(t *testing.T) {
	customErr1 := DefErr("err1", "错误1")
	customErr2 := DefErr("err2", "错误2")
	wrappedErr := fmt.Errorf("wrap: %w", customErr1)
	otherErr := errors.New("其他错误")

	tests := []struct {
		name    string
		err     error
		targets []*Err
		want    bool
	}{
		{
			name:    "直接匹配单个目标",
			err:     customErr1,
			targets: []*Err{customErr1},
			want:    true,
		},
		{
			name:    "不匹配单个目标",
			err:     customErr1,
			targets: []*Err{customErr2},
			want:    false,
		},
		{
			name:    "匹配多个目标中的一个",
			err:     customErr1,
			targets: []*Err{customErr2, customErr1},
			want:    true,
		},
		{
			name:    "不匹配多个目标",
			err:     otherErr,
			targets: []*Err{customErr1, customErr2},
			want:    false,
		},
		{
			name:    "匹配包装后的错误",
			err:     wrappedErr,
			targets: []*Err{customErr1},
			want:    true,
		},
		{
			name:    "不匹配包装后的其他错误",
			err:     wrappedErr,
			targets: []*Err{customErr2},
			want:    false,
		},
		{
			name:    "空目标列表",
			err:     customErr1,
			targets: []*Err{},
			want:    false,
		},
		{
			name:    "错误为nil",
			err:     nil,
			targets: []*Err{customErr1},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Is(tt.err, tt.targets...); got != tt.want {
				t.Errorf("Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

// 测试Wrapf函数
func TestWrapf(t *testing.T) {
	baseErr := errors.New("基础错误")
	biz := []string{"biz1", "biz2"}

	tests := []struct {
		name string
		err  error
		biz  []string
		msg  string
		args []any
		want string
	}{
		{
			name: "正常包装错误",
			err:  baseErr,
			biz:  biz,
			msg:  "操作失败: %s",
			args: []any{"参数错误"},
			want: "[biz1|biz2]操作失败: 参数错误 Err:基础错误",
		},
		{
			name: "空业务列表",
			err:  baseErr,
			biz:  []string{},
			msg:  "错误",
			args: nil,
			want: "[]错误 Err:基础错误",
		},
		{
			name: "无格式化参数",
			err:  baseErr,
			biz:  []string{"single"},
			msg:  "简单消息",
			args: nil,
			want: "[single]简单消息 Err:基础错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Wrapf(tt.err, tt.biz, tt.msg, tt.args...)
			if got.Error() != tt.want {
				t.Errorf("Wrapf() = %v, want %v", got, tt.want)
			}
		})
	}
}

// 测试NewBizErr函数
func TestNewBizErr(t *testing.T) {
	biz := []string{"testbiz"}

	tests := []struct {
		name string
		biz  []string
		msg  string
		args []any
		want string
	}{
		{
			name: "创建业务错误",
			biz:  biz,
			msg:  "业务错误: %d",
			args: []any{100},
			want: "[testbiz]业务错误: 100 Err:[biz_error]业务错误",
		},
		{
			name: "无参数业务错误",
			biz:  []string{"biz1", "biz2"},
			msg:  "操作失败",
			args: nil,
			want: "[biz1|biz2]操作失败 Err:[biz_error]业务错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewBizErr(tt.biz, tt.msg, tt.args...)
			if got.Error() != tt.want {
				t.Errorf("NewBizErr() = %v, want %v", got, tt.want)
			}
			// 验证是否是BizErr类型
			if !Is(got, BizErr) {
				t.Error("NewBizErr() 生成的错误不是BizErr类型")
			}
		})
	}
}

// 测试DefErr函数
func TestDefErr(t *testing.T) {
	tests := []struct {
		name string
		code string
		msg  string
		want *Err
	}{
		{
			name: "定义错误",
			code: "test_code",
			msg:  "test message",
			want: &Err{Code: "test_code", Msg: "test message"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DefErr(tt.code, tt.msg)
			if got.Code != tt.want.Code || got.Msg != tt.want.Msg {
				t.Errorf("DefErr() = %v, want %v", got, tt.want)
			}
		})
	}
}

// 测试预定义错误
func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      *Err
		wantCode string
		wantMsg  string
	}{
		{
			name:     "UnknownErr",
			err:      UnknownErr,
			wantCode: "unknown_error",
			wantMsg:  "未知错误",
		},
		{
			name:     "BizErr",
			err:      BizErr,
			wantCode: "biz_error",
			wantMsg:  "业务错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.wantCode {
				t.Errorf("Code = %v, want %v", tt.err.Code, tt.wantCode)
			}
			if tt.err.Msg != tt.wantMsg {
				t.Errorf("Msg = %v, want %v", tt.err.Msg, tt.wantMsg)
			}
		})
	}
}
