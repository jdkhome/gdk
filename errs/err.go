package errs

import (
	"errors"
	"fmt"
	"strings"
)

type Err struct {
	Code string
	Msg  string
}

func (e *Err) Error() string {
	return fmt.Sprintf("[%s]%s", e.Code, e.Msg)
}

func Is(err error, targets ...*Err) bool {
	var e *Err
	for _, target := range targets {
		if errors.As(err, &e) && e.Code == target.Code {
			return true
		}
	}
	return false
}

func Wrapf(err error, biz []string, msg string, args ...any) error {
	return fmt.Errorf("[%s]%s -> %w", strings.Join(biz, "|"), fmt.Sprintf(msg, args...), err)
}
