package errs

func DefErr(code, msg string) *Err {
	return &Err{
		Code: code,
		Msg:  msg,
	}
}

var (
	UnknownErr = DefErr("unknown_error", "未知错误")
	BizErr     = DefErr("biz_error", "业务错误")
	Timeout    = DefErr("timeout", "超时")
)

func NewBizErr(biz []string, msg string, args ...any) error {
	return Wrapf(BizErr, biz, msg, args...)
}

func NewTimeout(biz []string, msg string, args ...any) error {
	return Wrapf(Timeout, biz, msg, args...)
}
