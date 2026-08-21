# enums

通用的泛型枚举构建包，提供一套类型安全、可复用、带 code 与 value 的枚举定义方式。

## 内容

- `Value`：枚举值的基础结构，包含 `Name` 与 `Desc`。
- `Member[V]`：枚举成员，由 `Code` 与 `Value *V` 组成。
- `Enum[M, V]`：构建完成的枚举集合，支持 `Members`、`CheckMember`、`GetByCode`。
- `Builder[M, V]`：枚举构建器，用于按 code 添加成员并最终 `Build`。

## 使用示例

```go
package main

import "github.com/jdkhome/gdk/enums"

// 自定义枚举类型
type Color enums.Member[enums.Value]

var (
    colorBuilder = enums.NewBuilder[Color]()
    Red          = colorBuilder.Add("red", enums.NewEnumValueWithDesc("红色", "红颜色"))
    Blue         = colorBuilder.Add("blue", enums.NewEnumValue("蓝色"))
    Colors       = colorBuilder.Build()
)

func main() {
    if color, ok := Colors.GetByCode("red"); ok {
        println(color.Value.Name)
    }
}
```

## 注意事项

- 重复添加相同 code 会触发 panic。
- `Build` 之后不能再继续 `Add`，否则会 panic。
