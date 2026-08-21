package di

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenerateGenericsAndList 端到端验证泛型接口绑定、泛型 provider 与 list 注入。
func TestGenerateGenericsAndList(t *testing.T) {
	dir := t.TempDir()

	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("go.mod", "module example.com/testdi\n\ngo 1.24\n\nrequire github.com/jdkhome/gdk v0.0.0\nreplace github.com/jdkhome/gdk => "+root+"\n")

	write("providers.go", `package testdi

import "github.com/jdkhome/gdk/di"

type Handler interface{ Handle() string }

type AHandler struct{}
func (AHandler) Handle() string { return "A" }

type BHandler struct{}
func (BHandler) Handle() string { return "B" }

type Repository[T any] interface{ Get(id int) (T, error) }

type User struct{ Name string }
type UserRepo struct{}
func (UserRepo) Get(id int) (User, error) { return User{Name: "alice"}, nil }

type Box[T any] struct{ Value T }
func NewBox[T any](v T) *Box[T] { return &Box[T]{Value: v} }
func ProvideUser() User { return User{Name: "bob"} }

var Set = di.NewSet(
	di.Struct(new(AHandler), "*"),
	di.Struct(new(BHandler), "*"),
	di.Bind(new(Handler), new(AHandler)),
	di.Bind(new(Handler), new(BHandler)),
	di.Struct(new(UserRepo), "*"),
	di.Bind(new(Repository[User]), new(UserRepo)),
)

var BoxSet = di.NewSet(
	ProvideUser,
	NewBox[User],
)

var BoxStructSet = di.NewSet(
	ProvideUser,
	di.Struct(new(Box[User]), "*"),
)
`)

	write("injector.go", `//go:build diinject

package testdi

import "github.com/jdkhome/gdk/di"

func InitHandlers() []Handler {
	di.Build(Set)
	return nil
}

func InitRepo() Repository[User] {
	di.Build(Set)
	return nil
}

func InitBox() *Box[User] {
	di.Build(BoxSet)
	return nil
}

func InitStructBox() *Box[User] {
	di.Build(BoxStructSet)
	return nil
}
`)

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy 失败: %v\n%s", err, out)
	}

	outs, errs := Generate(context.Background(), dir, os.Environ(), []string{"."}, nil)
	if len(errs) > 0 {
		t.Fatalf("Generate 返回错误: %v", errs)
	}
	if len(outs) != 1 {
		t.Fatalf("期望 1 个输出，得到 %d", len(outs))
	}
	if len(outs[0].Errs) > 0 {
		t.Fatalf("生成失败: %v", outs[0].Errs)
	}

	content := string(outs[0].Content)
	checks := []string{
		"[]Handler{aHandler, bHandler}", // list 注入收集两个实现
		"Repository[User]",              // 泛型接口绑定
		"userRepo := UserRepo{}",
		"NewBox[User](user)", // 泛型函数 provider
		"&Box[User]{",        // 泛型结构体 provider
	}
	for _, c := range checks {
		if !strings.Contains(content, c) {
			t.Errorf("生成内容缺少 %q:\n%s", c, content)
		}
	}
}
