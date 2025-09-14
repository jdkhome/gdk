package logs

import "github.com/jdkhome/gdk/enums"

type Level enums.Member[LevelValue]
type LevelValue struct {
	enums.Value
	level int
}

var (
	levelEnumBuilder = enums.NewBuilder[Level]()
	Level_Debug      = levelEnumBuilder.Add("debug", LevelValue{Value: enums.NewEnumValue("DEBUG"), level: 0})
	Level_Info       = levelEnumBuilder.Add("info", LevelValue{Value: enums.NewEnumValue("INFO"), level: 1})
	Level_Warn       = levelEnumBuilder.Add("warn", LevelValue{Value: enums.NewEnumValue("WARN"), level: 2})
	Level_Error      = levelEnumBuilder.Add("error", LevelValue{Value: enums.NewEnumValue("ERROR"), level: 3})
	Levels           = levelEnumBuilder.Build()
)
