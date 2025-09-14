package ctxs

import "github.com/jdkhome/gdk/enums"

type Key enums.Member[enums.Value]

var (
	keyEnumBuilder = enums.NewBuilder[Key]()
	Trace          = keyEnumBuilder.Add("CTX#TRACE", enums.NewEnumValue("trace消息"))
	Keys           = keyEnumBuilder.Build()
)
