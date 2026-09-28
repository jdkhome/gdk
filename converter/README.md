# converter

对象转换器包，提供泛型转换接口与转换方法，用于将一种类型的对象（或对象列表）转换为另一种类型。

## 内容

- `Converter[FROM, TO]`：转换器接口，定义 `DoConverter(ctx, from)` 方法。
- `Convert[FROM, TO]`：将单个对象转换为目标类型。
- `ConvertList[FROM, TO]`：将对象列表转换为目标类型列表。
- `NilFromErr`：转换源对象为空的错误定义，可通过 `errs.Is` 判断。

## 使用示例

```go
package main

import (
    "context"
    "strconv"

    "github.com/jdkhome/gdk/converter"
)

// IntToStringConverter 将 int 转换为 string 的转换器。
type IntToStringConverter struct{}

func (c *IntToStringConverter) DoConverter(_ context.Context, from int) (string, error) {
    return strconv.Itoa(from), nil
}

func main() {
    c := &IntToStringConverter{}
    ctx := context.Background()

    to, err := converter.Convert(ctx, c, 42)
    // to == "42"

    list, err := converter.ConvertList(ctx, c, []int{1, 2, 3})
    // list == []string{"1", "2", "3"}
}
```

## 注意事项

- `Convert` 在源对象为 nil 时返回 `NilFromErr` 错误；`ConvertList` 在任一元素为 nil 时同样返回该错误。
- `ConvertList` 在任一元素转换失败时立即返回错误，且不返回已转换的部分结果。
- 若在并发场景复用同一个转换器对象，需由转换器实现方自行保证并发安全。
