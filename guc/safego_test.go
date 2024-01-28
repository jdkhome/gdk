package guc

import (
	"context"
	"fmt"
	"github.com/jdkhome/gdk/error_code"
	"github.com/jdkhome/gdk/exception"
	"testing"
	"time"
)

func TestSafeGo(t *testing.T) {
	GoRunnable(context.TODO(), func(ctx context.Context) {
		i := 0
		for {
			i++
			fmt.Printf("GoRunnable: i= %d\n", i)
			time.Sleep(time.Second)
			if i == 5 {
				exception.Throw(error_code.Error, "GoRunnable 抛异常")
			}
		}
	})

	result := GoCallable[int](context.TODO(), func(ctx context.Context) int {
		i := 0
		for {
			i++
			fmt.Printf("GoCallable: i= %d\n", i)
			time.Sleep(time.Second)
			if i == 10 {
				break
			}
		}
		return i
	})

	i := 0
	for {
		i++
		fmt.Printf("Main: i= %d\n", i)
		time.Sleep(time.Second)
		if i == 4 {
			break
		}
	}
	fmt.Printf("Main over\n")
	fmt.Printf("GoCallable result:%d\n", result.GetResult())
}
