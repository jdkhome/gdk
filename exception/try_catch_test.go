package exception

import (
	"errors"
	"fmt"
	"github.com/jdkhome/gdk/error_code"
	"testing"
)

func TestTryCatchBiz(t *testing.T) {
	TryCatch(
		func() {
			Throw(error_code.Error, "故意抛 BizException")
		},
		Catch{Biz, func(err any) {
			fmt.Printf("捕获到%v异常: %v\n", Biz, err)
		}},
		Catch{Err, func(err any) {
			fmt.Printf("捕获到%v异常: %v\n", Err, err)
		}},
		Catch{Any, func(err any) {
			fmt.Printf("捕获到%v异常: %v\n", Any, err)
		}},
	)
}

func TestTryCatchErr(t *testing.T) {
	TryCatch(
		func() {
			panic(errors.New("故意抛 error"))
		},
		Catch{Biz, func(err any) {
			fmt.Printf("捕获到%v异常: %v\n", Biz, err)
		}},
		Catch{Err, func(err any) {
			fmt.Printf("捕获到%v异常: %v\n", Err, err)
		}},
		Catch{Any, func(err any) {
			fmt.Printf("捕获到%v异常: %v\n", Any, err)
		}},
	)
}

func TestTryCatchAny(t *testing.T) {
	TryCatch(
		func() {
			panic("故意抛 any")
		},
		Catch{Biz, func(err any) {
			fmt.Printf("捕获到%v异常: %v\n", Biz, err)
		}},
		Catch{Err, func(err any) {
			fmt.Printf("捕获到%v异常: %v\n", Err, err)
		}},
		Catch{Any, func(err any) {
			fmt.Printf("捕获到%v异常: %v\n", Any, err)
		}},
	)
}
