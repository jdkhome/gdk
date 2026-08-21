package example

import "github.com/jdkhome/gdk/di"

// Handler 是一个普通接口，用于演示 list 注入。
type Handler interface {
	Handle() string
}

type AHandler struct{}

func (AHandler) Handle() string { return "A" }

type BHandler struct{}

func (BHandler) Handle() string { return "B" }

// HandlerSet 对同一接口 Handler 进行两次 Bind，用于自动收集。
var HandlerSet = di.NewSet(
	di.Struct(new(AHandler), "*"),
	di.Struct(new(BHandler), "*"),
	di.Bind(new(Handler), new(AHandler)),
	di.Bind(new(Handler), new(BHandler)),
)

// Repository 是一个泛型接口，用于演示泛型接口绑定。
type Repository[T any] interface {
	Get(id int) (T, error)
}

type User struct {
	Name string
}

type UserRepo struct{}

func (UserRepo) Get(id int) (User, error) {
	return User{Name: "alice"}, nil
}

// RepoSet 将实例化后的泛型接口 Repository[User] 绑定到 UserRepo。
var RepoSet = di.NewSet(
	di.Struct(new(UserRepo), "*"),
	di.Bind(new(Repository[User]), new(UserRepo)),
)

// Box 是一个泛型结构体，NewBox 是一个泛型 provider 函数。
type Box[T any] struct {
	Value T
}

func NewBox[T any](v T) *Box[T] {
	return &Box[T]{Value: v}
}

func ProvideUser() User {
	return User{Name: "bob"}
}

// BoxSet 演示泛型 provider：泛型函数显式实例化为 NewBox[User] 后作为 provider。
var BoxSet = di.NewSet(
	ProvideUser,
	NewBox[User],
)

// BoxStructSet 演示泛型结构体 provider：di.Struct(new(Box[User]), "*")。
var BoxStructSet = di.NewSet(
	ProvideUser,
	di.Struct(new(Box[User]), "*"),
)
