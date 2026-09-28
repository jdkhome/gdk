package converter

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/jdkhome/gdk/errs"
)

// intToStringConverter 测试用转换器，将 int 转换为 string，负数转换失败。
type intToStringConverter struct{}

func (c *intToStringConverter) DoConverter(_ context.Context, from int) (string, error) {
	if from < 0 {
		return "", errors.New("不支持负数")
	}
	return strconv.Itoa(from), nil
}

// ptrToStringConverter 测试用转换器，将 *int 转换为 string。
type ptrToStringConverter struct{}

func (c *ptrToStringConverter) DoConverter(_ context.Context, from *int) (string, error) {
	return strconv.Itoa(*from), nil
}

func TestConvert(t *testing.T) {
	c := &intToStringConverter{}

	tests := []struct {
		name    string
		from    int
		want    string
		wantErr bool
	}{
		{name: "正常转换", from: 42, want: "42", wantErr: false},
		{name: "负数转换报错", from: -1, want: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Convert(context.Background(), c, tt.from)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Convert() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Convert() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvert_NilFrom(t *testing.T) {
	c := &ptrToStringConverter{}
	var from *int

	_, err := Convert(context.Background(), c, from)
	if err == nil {
		t.Fatal("from 为 nil 时应返回错误")
	}
	if !errs.Is(err, NilFromErr) {
		t.Errorf("错误类型应为 NilFromErr，实际为 %v", err)
	}
}

func TestConvertList(t *testing.T) {
	c := &intToStringConverter{}

	tests := []struct {
		name     string
		fromList []int
		want     []string
		wantErr  bool
	}{
		{name: "正常转换列表", fromList: []int{1, 2, 3}, want: []string{"1", "2", "3"}, wantErr: false},
		{name: "空列表", fromList: []int{}, want: []string{}, wantErr: false},
		{name: "包含负数转换报错", fromList: []int{1, -2}, want: nil, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertList(context.Background(), c, tt.fromList)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ConvertList() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ConvertList() len = %v, want %v", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ConvertList()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestConvertList_NilElement(t *testing.T) {
	c := &ptrToStringConverter{}
	v := 1

	_, err := ConvertList(context.Background(), c, []*int{&v, nil})
	if err == nil {
		t.Fatal("列表含 nil 元素时应返回错误")
	}
	if !errs.Is(err, NilFromErr) {
		t.Errorf("错误类型应为 NilFromErr，实际为 %v", err)
	}
}
