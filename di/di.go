// Copyright 2018 The Wire Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package di 提供编译期依赖注入的代码生成指令，由 github.com/google/wire 移植并增强。
//
// 本包中的指令作为 di 代码生成工具的输入。di 分析的入口是「注入器函数」：
// 即函数体中仅包含一次 Build 调用的函数模板。Build 的参数描述一组 provider，
// di 代码生成工具会构建这些 provider 输出类型的有向无环图，并利用 provider
// 集合来实例化所需的类型，最终填充函数模板。
package di

// ProviderSet 是收集一组 provider 的标记类型。
type ProviderSet struct{}

// NewSet 创建一个新的 provider 集合，包含其参数中的 provider。每个参数可以是：
// 一个函数值、一个 provider 集合、一次 Struct 调用、一次 Bind 调用、
// 一次 Value 调用、一次 InterfaceValue 调用或一次 FieldsOf 调用。
//
// 向 NewSet 传入函数值表示：该函数的第一个返回值类型将通过调用该函数来提供。
// 函数的参数将来自其参数类型对应的 provider。因此，函数的所有参数必须具有
// 互不相同的类型。函数可选地在最后一个返回值返回 error，在第二个返回值返回
// 清理函数。清理函数必须为 func() 类型，并保证先于其任何输入的清理函数被调用。
// 如果任意 provider 返回 error，注入器函数将调用所有相应的清理函数，并从
// 注入器函数返回该 error。
//
// 向 NewSet 传入 ProviderSet，等同于将该集合的内容直接作为 NewSet 的参数。
//
// 传入本包其他函数调用结果的行为，在各函数自己的文档注释中说明。
//
// 为兼容旧版，向 NewSet 传入类型 S 的结构体值，表示 S 和 *S 都将通过创建
// 一个适当类型的新值（即用字段类型对应的 provider 填充 S 的每个字段）来提供。
// 此形式已废弃：新的 provider 集合应使用 di.Struct。
func NewSet(...interface{}) ProviderSet {
	return ProviderSet{}
}

// Build 放置在注入器函数模板的函数体中，用于声明要使用的 provider。di 代码
// 生成工具会填充该函数的实现。Build 的参数解释方式与 NewSet 相同：它们决定
// 提供给 di 依赖图的 provider 集合。Build 返回一个可传给 panic() 调用的错误信息。
//
// 注入器函数的参数被用作依赖图中的输入。
//
// 与传入 NewSet 的 provider 函数类似，第一个返回值是注入器函数的输出，可选的
// 第二个返回值是清理函数，可选的最后一个返回值是 error。如果注入器函数
// provider 集合中的任意 provider 函数返回 error 或清理函数，则注入器函数模板
// 中必须存在对应的返回值。
//
// 示例：
//
//	func injector(ctx context.Context) (*sql.DB, error) {
//		di.Build(otherpkg.FooSet, myProviderFunc)
//		return nil, nil
//	}
//
//	func injector(ctx context.Context) (*sql.DB, error) {
//		panic(di.Build(otherpkg.FooSet, myProviderFunc))
//	}
func Build(...interface{}) string {
	return "未生成实现，请运行 di"
}

// Binding 将一个接口映射到一个具体类型。
type Binding struct{}

// Bind 声明：应使用一个具体类型来满足对 iface 类型的依赖。iface 必须是指向
// 接口类型的指针，to 必须是指向具体类型的指针。
//
// 示例：
//
//	type Fooer interface {
//		Foo()
//	}
//
//	type MyFoo struct{}
//
//	func (MyFoo) Foo() {}
//
//	var MySet = di.NewSet(
//		di.Struct(new(MyFoo))
//		di.Bind(new(Fooer), new(MyFoo)))
func Bind(iface, to interface{}) Binding {
	return Binding{}
}

// bindToUsePointer 由 di 工具检测，用于指示 Bind 的第二个参数应使用指针。
// 详见 https://github.com/google/wire/issues/120。
const bindToUsePointer = true

// ProvidedValue 是一个被复制到生成后注入器中的表达式。
type ProvidedValue struct{}

// Value 绑定一个表达式，以提供该表达式的类型。
// 该表达式不能是接口值；若是接口值请使用 InterfaceValue。
//
// 示例：
//
//	var MySet = di.NewSet(di.Value([]string(nil)))
func Value(interface{}) ProvidedValue {
	return ProvidedValue{}
}

// InterfaceValue 绑定一个表达式，以提供特定的接口类型。
// 第一个参数是指向要提供的接口的指针，第二个参数是类型实现了该接口的实际变量值。
//
// 示例：
//
//	var MySet = di.NewSet(di.InterfaceValue(new(io.Reader), os.Stdin))
func InterfaceValue(typ interface{}, x interface{}) ProvidedValue {
	return ProvidedValue{}
}

// StructProvider 表示一个具名结构体。
type StructProvider struct{}

// Struct 指定：给定结构体类型将通过填充指定名称的字段来提供。
//
// 第一个参数必须是指向结构体类型的指针。对于结构体类型 Foo，di 会使用字段
// 填充来提供 Foo 和 *Foo。其余参数是要填充的字段名。特殊情况下，如果只给一个
// 名称 "*"，则结构体的所有字段都会被填充。
//
// 例如：
//
//	type S struct {
//	  MyFoo *Foo
//	  MyBar *Bar
//	}
//	var Set = di.NewSet(di.Struct(new(S), "MyFoo")) -> 仅注入 S.MyFoo
//	var Set = di.NewSet(di.Struct(new(S), "*")) -> 注入所有字段
func Struct(structType interface{}, fieldNames ...string) StructProvider {
	return StructProvider{}
}

// StructFields 是结构体字段的集合。
type StructFields struct{}

// FieldsOf 声明：给定结构体类型中指定名称的字段将用于提供这些字段的类型。
// structType 参数必须是指向结构体的指针，或指向「指向结构体的指针」的指针。
//
// 以下示例将分别使用 S.MyFoo 和 S.MyBar 提供 Foo 和 Bar：
//
//	type S struct {
//		MyFoo Foo
//		MyBar Bar
//	}
//
//	func NewStruct() S { /* ... */ }
//	var Set = di.NewSet(di.FieldsOf(new(S), "MyFoo", "MyBar"))
//
//	或
//
//	func NewStruct() *S { /* ... */ }
//	var Set = di.NewSet(di.FieldsOf(new(*S), "MyFoo", "MyBar"))
//
//	如果 structType 参数是指向「指向结构体的指针」的指针，则 FieldsOf 还会额外
//	提供指向每个字段类型的指针（例如上面示例中的 *Foo 和 *Bar）。
func FieldsOf(structType interface{}, fieldNames ...string) StructFields {
	return StructFields{}
}
