package exception

import (
	"fmt"
	"github.com/jdkhome/gdk/error_code"
)

type BizException struct {
	Code     error_code.ErrorCode
	Msg      string
	MoreInfo string
}

func (be BizException) Error() string {
	return be.String()
}

func (be BizException) String() string {
	str := fmt.Sprintf("[%v/%v]%v", be.Code.GetCode(), be.Code.GetName(), be.Msg)
	if be.MoreInfo != "" {
		str += fmt.Sprintf(" : %v", be.MoreInfo)
	}
	return str
}

func Throw(code error_code.ErrorCode, msg string) {
	panic(BizException{Code: code, Msg: msg})
}

func ThrowWithMoreInfo(code error_code.ErrorCode, msg string, moreInfo string) {
	panic(BizException{Code: code, Msg: msg, MoreInfo: moreInfo})
}
