# di

编译期依赖注入代码生成工具，基于 [github.com/google/wire](https://github.com/google/wire) 移植并增强。

与 wire 相同的编译期代码生成思路：在「注入器函数」（injector）中调用 `di.Build(...)` 声明依赖，运行 `di gen` 后生成 `di_gen.go` 实现这些注入器。相比 wire，额外支持：

- **泛型接口绑定**：`di.Bind(new(Repo[User]), new(UserRepo))`
- **泛型 provider**：泛型函数 / 泛型结构体可直接作为 provider（显式实例化，如 `NewBox[User]`、`di.Struct(new(Box[User]), "*")`）
- **list 注入**：对同一类型多次 `Bind`，注入 `[]T` 时自动收集全部实现

## 安装

```bash
go get github.com/jdkhome/gdk/di
```

## 快速开始

1. 定义 provider 与注入器：

```go
// providers.go
package app

import "github.com/jdkhome/gdk/di"

type Handler interface{ Handle() string }

type AHandler struct{}
func (AHandler) Handle() string { return "A" }

type BHandler struct{}
func (BHandler) Handle() string { return "B" }

var HandlerSet = di.NewSet(
    di.Struct(new(AHandler), "*"),
    di.Struct(new(BHandler), "*"),
    di.Bind(new(Handler), new(AHandler)),
    di.Bind(new(Handler), new(BHandler)),
)
```

```go
//go:build diinject

package app

import "github.com/jdkhome/gdk/di"

// 注入 []Handler，自动收集所有 Handler 实现。
func InitHandlers() []Handler {
    di.Build(HandlerSet)
    return nil
}
```

2. 生成代码：

```bash
go run github.com/jdkhome/gdk/di/cmd/di gen ./...
```

生成 `di_gen.go`（`//go:build !diinject`）：

```go
func InitHandlers() []Handler {
    aHandler := AHandler{}
    bHandler := BHandler{}
    v := []Handler{aHandler, bHandler}
    return v
}
```

## 泛型支持

```go
type Repository[T any] interface {
    Get(id int) (T, error)
}

type User struct{ Name string }
type UserRepo struct{}
func (UserRepo) Get(id int) (User, error) { return User{}, nil }

// 泛型接口绑定：实例化后的 Repository[User] 绑定到 UserRepo。
var RepoSet = di.NewSet(
    di.Struct(new(UserRepo), "*"),
    di.Bind(new(Repository[User]), new(UserRepo)),
)

// 泛型 provider：泛型函数显式实例化后作为 provider。
func NewBox[T any](v T) *Box[T] { return &Box[T]{Value: v} }
type Box[T any] struct{ Value T }

var BoxSet = di.NewSet(
    di.NewSet(NewBox[User]),
)

// 泛型结构体 provider。
var BoxStructSet = di.NewSet(
    di.Struct(new(Box[User]), "*"),
)
```

## 注意

- 泛型 provider 需显式实例化（Go 不允许把未实例化的泛型函数作为值传递），例如 `NewBox[User]`。
- 注入 `[]T` 会收集该类型的所有 provider；若对存在多个 provider 的类型做单值注入会报错。
- 生成文件带有 `//go:build !diinject` 构建标签，注入器模板文件应使用 `//go:build diinject`。

## 完整示例

见 [example](example/) 目录。
