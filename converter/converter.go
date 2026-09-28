package converter

import (
	"context"
	"reflect"

	"github.com/jdkhome/gdk/errs"
)

// NilFromErr 表示转换源对象为空的错误，可通过 errs.Is(err, NilFromErr) 判断。
var NilFromErr = errs.DefErr("converter_nil_from", "转换源对象为空")

// Converter 定义对象转换器接口，用于将 FROM 类型对象转换为 TO 类型对象。
type Converter[FROM any, TO any] interface {
	// DoConverter 将单个 FROM 类型对象转换为 TO 类型对象。
	DoConverter(ctx context.Context, from FROM) (TO, error)
}

// Convert 使用指定的转换器将单个对象转换为目标类型。
// 当 from 为 nil 时返回 NilFromErr 错误。
func Convert[FROM any, TO any](ctx context.Context, converter Converter[FROM, TO], from FROM) (TO, error) {
	var zero TO
	if isNil(from) {
		return zero, errs.Wrapf(NilFromErr, []string{"converter"}, "转换源对象为空")
	}
	return converter.DoConverter(ctx, from)
}

// ConvertList 使用指定的转换器将对象切片转换为目标类型切片。
// 任一元素为 nil 或转换失败时立即返回错误，且不返回已转换的部分结果。
func ConvertList[FROM any, TO any](ctx context.Context, converter Converter[FROM, TO], fromList []FROM) ([]TO, error) {
	toList := make([]TO, 0, len(fromList))
	for _, from := range fromList {
		if isNil(from) {
			return nil, errs.Wrapf(NilFromErr, []string{"converter"}, "转换源对象为空")
		}
		to, err := converter.DoConverter(ctx, from)
		if err != nil {
			return nil, err
		}
		toList = append(toList, to)
	}
	return toList, nil
}

// isNil 判断 any 值是否为 nil，覆盖 nil 指针、map、slice、func、chan 与 interface。
func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan, reflect.Interface:
		return rv.IsNil()
	default:
		return false
	}
}
