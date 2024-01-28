package assert

import (
	"github.com/jdkhome/gdk/error_code"
	"testing"
)

func TestIsNil(t *testing.T) {
	IsNil(error_code.Error, "value应该为nil", nil)
}
