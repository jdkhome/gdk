package error_code

import "fmt"

type ErrorCode int

const (
	ParamError ErrorCode = iota
	Error
)

func init() {
	DefErrorCode(ParamError, ErrorCodeInfo{code: "400", name: "参数错误"})
	DefErrorCode(Error, ErrorCodeInfo{code: "500", name: "错误"})
}

type ErrorCodeInfo struct {
	code string
	name string
}

var (
	errorCodeInfoMap = make(map[ErrorCode]ErrorCodeInfo)
	errorCodeCodeMap = make(map[string]ErrorCode)
)

func DefErrorCode(enum ErrorCode, info ErrorCodeInfo) {
	if _, isDef := errorCodeInfoMap[enum]; isDef {
		panic(fmt.Sprintf("错误吗已被定义，检查号段是否重复 val=%v", enum))
	}
	errorCodeInfoMap[enum] = info

	if _, isDef := errorCodeCodeMap[info.code]; isDef {
		panic(fmt.Sprintf("错误吗code重复 code=%v", info.code))
	}
	errorCodeCodeMap[info.code] = enum
}

func (e ErrorCode) GetCode() string { return errorCodeInfoMap[e].code }
func (e ErrorCode) GetName() string { return errorCodeInfoMap[e].name }

func GetByCode(code string) ErrorCode {
	if _, isDef := errorCodeCodeMap[code]; !isDef {
		panic(fmt.Sprintf("错误吗code未定义 code=%v", code))
	}
	return errorCodeCodeMap[code]
}

func GetAll() []ErrorCode {
	values := make([]ErrorCode, 0, len(errorCodeCodeMap))
	for _, v := range errorCodeCodeMap {
		values = append(values, v)
	}
	return values
}
