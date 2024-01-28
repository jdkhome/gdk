package exception

import (
	"fmt"
	"reflect"
)

var Biz = reflect.TypeOf((*BizException)(nil)).Elem()
var Err = reflect.TypeOf((*error)(nil)).Elem()
var Any = reflect.TypeOf((*any)(nil)).Elem()

type Catch struct {
	Type reflect.Type
	Func func(any)
}

func TryCatch(try func(), catchList ...Catch) {
	defer func() {
		if r := recover(); r != nil {
			thisType := reflect.TypeOf(r)

			// 按顺序检查，看那个catch能最先处理
			for _, catch := range catchList {
				// 判断直接是否是某类型 或者 实现了接口
				if (thisType == catch.Type) || (catch.Type.Kind() == reflect.Interface && thisType.Implements(catch.Type)) {
					catch.Func(r)
					return
				}
			}
			// 兜不住的 再次抛出
			fmt.Printf("can not catch panic:%v\n", r)
			panic(r)
		}
	}()
	try()
}
