package logs

import (
	"context"
)

//type OutputType enums.Member[enums.Value]

//var (
//	outputTypeEnumBuilder = enums.NewBuilder[OutputType]()
//	Output_File           = outputTypeEnumBuilder.Add("file", enums.NewEnumValue("输出到文件"))
//	Output_Console        = outputTypeEnumBuilder.Add("console", enums.NewEnumValue("输出到控制台"))
//	Outputs               = outputTypeEnumBuilder.Build()
//)

type Output interface {
	PushLog(ctx context.Context, level Level, content string) (err error)
}
