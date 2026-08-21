# i18n

国际化相关的轻量类型定义。

## 内容

- `Str`：以 `language.Tag` 为键的多语言字符串映射，本质是 `map[language.Tag]string`。

## 使用示例

```go
package main

import (
    "github.com/jdkhome/gdk/model/i18n"
    "golang.org/x/text/language"
)

func main() {
    msg := i18n.Str{
        language.Chinese: "你好",
        language.English: "Hello",
    }
    println(msg[language.English])
}
```
