package assert

import (
	"fmt"
	"gdk/error_code"
	"gdk/exception"
	"strings"
)

func IsNil(code error_code.ErrorCode, msg string, value *any) {
	if value != nil {
		exception.ThrowWithMoreInfo(code, msg, fmt.Sprintf("Assert IsNil but value != nil"))
	}
}

func IsNotNil(code error_code.ErrorCode, msg string, value *any) {
	if value == nil {
		exception.ThrowWithMoreInfo(code, msg, fmt.Sprintf("Assert IsNotNil but value = nil"))
	}
}

func IsTrue(code error_code.ErrorCode, msg string, value bool) {
	if !value {
		exception.ThrowWithMoreInfo(code, msg, fmt.Sprintf("Assert IsTrue but value = %v", value))
	}
}

func IsFalse(code error_code.ErrorCode, msg string, value bool) {
	if value {
		exception.ThrowWithMoreInfo(code, msg, fmt.Sprintf("Assert IsFalse but value = %v", value))
	}
}

func IsEqual(code error_code.ErrorCode, msg string, values ...any) {
	for i := 1; i < len(values); i++ {
		if values[i-1] != values[i] {
			exception.ThrowWithMoreInfo(code, msg, fmt.Sprintf("Assert IsEqual but value[%d]=%v value[%d]=%v", i-1, values[i-1], i, values[i]))
		}
	}
}

func IsNotEqual(code error_code.ErrorCode, msg string, values ...any) {
	for i := 1; i < len(values); i++ {
		if values[i-1] == values[i] {
			exception.ThrowWithMoreInfo(code, msg, fmt.Sprintf("Assert IsNotEqual but value[%d]=%v value[%d]=%v", i-1, values[i-1], i, values[i]))
		}
	}
}

func IsEmpty(code error_code.ErrorCode, msg string, value string) {
	if len(value) != 0 {
		exception.ThrowWithMoreInfo(code, msg, fmt.Sprintf("Assert IsEmpty but len(value)=%v", len(value)))
	}
}

func IsNotEmpty(code error_code.ErrorCode, msg string, value string) {
	if len(value) == 0 {
		exception.ThrowWithMoreInfo(code, msg, fmt.Sprintf("Assert IsNotEmpty but len(value)=%v", len(value)))
	}
}

func IsBlank(code error_code.ErrorCode, msg string, value string) {
	trim := strings.TrimSpace(value)
	if len(trim) != 0 {
		exception.ThrowWithMoreInfo(code, msg, fmt.Sprintf("Assert IsBlank but len(trim(value))=%v", len(trim)))
	}
}

func IsNotBlank(code error_code.ErrorCode, msg string, value string) {
	trim := strings.TrimSpace(value)
	if len(trim) == 0 {
		exception.ThrowWithMoreInfo(code, msg, fmt.Sprintf("Assert IsNotBlank but len(trim(value))=%v", len(trim)))
	}
}
