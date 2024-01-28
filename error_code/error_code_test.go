package error_code

import (
	"fmt"
	"testing"
)

func TestGetByCode(t *testing.T) {
	fmt.Println(GetByCode(Error.GetCode()).GetName())
}

func TestGetAll(t *testing.T) {
	for _, errorCode := range GetAll() {
		fmt.Printf("code:%v name:%v\n", errorCode.GetCode(), errorCode.GetName())
	}
}
