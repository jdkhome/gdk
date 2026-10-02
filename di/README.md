# di

编译期依赖注入代码生成工具，基于 [github.com/google/wire](https://github.com/google/wire) 移植并增强。

与 wire 相同的编译期代码生成思路：在「注入器函数」（injector）中调用 `di.Build(...)` 声明依赖，运行生成器后生成 `di_gen.go` 实现这些注入器。相比 wire，额外支持：

- **泛型接口绑定**：`di.Bind(new(Repo[User]), new(UserRepo))`
- **泛型 provider**：泛型函数 / 泛型结构体可直接作为 provider（显式实例化，如 `NewBox[User]`、`di.Struct(new(Box[User]), "*")`）
- **list 注入**：对同一类型多次 `Bind`，注入 `[]T` 时自动收集已绑定且有具体 provider 的实现（泛型函数 provider 的限制见下文）

## 安装

在调用方 Go 模块中添加依赖：

```bash
go get github.com/jdkhome/gdk/di
```

可直接使用下文的 `go run -mod=mod`（仅 `go get .../di` 不一定补齐命令包依赖的 `go.sum`，`-mod=mod` 允许 Go 按需更新模块依赖与校验和），或安装命令行工具：

```bash
go install github.com/jdkhome/gdk/di/cmd/di@latest
di ./...
```

安装后的 `di` 需位于 PATH 中。CLI 没有 `gen` 子命令，位置参数直接是 Go 包路径；不传路径时处理当前包。支持 `-header_file`、`-output_file_prefix`、`-tags`，选项需放在包路径之前。

## 快速开始

1. 定义 provider 与注入器（分别保存为 `providers.go` 与 `injector.go`）：

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

// 注入 []Handler，自动收集 HandlerSet 中绑定的 AHandler 与 BHandler。
func InitHandlers() []Handler {
    di.Build(HandlerSet)
    return nil
}
```

2. 在调用方模块中生成代码：

```bash
go run -mod=mod github.com/jdkhome/gdk/di/cmd/di ./...
```

生成 `di_gen.go`（`//go:build !diinject`）中的函数：

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
func ProvideUser() User { return User{Name: "bob"} }

var BoxSet = di.NewSet(
    ProvideUser,
    NewBox[User],
)

// 泛型结构体 provider；与 BoxSet 二选一使用。
var BoxStructSet = di.NewSet(
    ProvideUser,
    di.Struct(new(Box[User]), "*"),
)
```

`BoxSet` 可用于单值注入 `*Box[User]`，但不能自动收集为 `[]*Box[User]`。需要列表时，可改用上面的 `BoxStructSet`，或用返回 `*Box[User]` 的非泛型函数包装 `NewBox[User]`，替换 `BoxSet` 中的泛型函数 provider。

## 注意

- 泛型 provider 需显式实例化（Go 不允许把未实例化的泛型函数作为值传递），例如 `NewBox[User]`。
- 没有直接提供 `[]T` 的 provider 或注入器参数时，注入 `[]T` 会收集已登记到元素类型 `T` 的具体 provider；没有可收集的元素 provider 时会报错，不会生成空列表。泛型函数 provider 单独保存，即使在集合中写成 `NewBox[User]`，列表收集也不会为元素类型触发泛型实例化。对存在多个具体 provider 的类型做单值注入会报错。
- `di.Bind` 的具体类型也需有 provider；若接口由 `*UserRepo` 实现，应使用 `new(*UserRepo)` 绑定。
- 生成文件带有 `//go:build !diinject` 构建标签，注入器模板文件应使用 `//go:build diinject`。

## 完整示例

见 [example](example/) 目录；在 gdk 仓库根目录可运行 `go run -mod=mod ./di/cmd/di ./di/example` 重新生成示例。
