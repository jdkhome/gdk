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

package di

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/types/typeutil"
)

// providerSetSrc 记录由 ProviderSet 提供的类型的来源。
// 这些字段中恰好有一个会被设置。
type providerSetSrc struct {
	Provider    *Provider
	Binding     *IfaceBinding
	Value       *Value
	Import      *ProviderSet
	InjectorArg *InjectorArg
	Field       *Field
}

// description 返回描述 p 来源的字符串（含行号）。
func (p *providerSetSrc) description(fset *token.FileSet, typ types.Type) string {
	quoted := func(s string) string {
		if s == "" {
			return ""
		}
		return fmt.Sprintf("%q ", s)
	}
	switch {
	case p.Provider != nil:
		kind := "provider"
		if p.Provider.IsStruct {
			kind = "struct provider"
		}
		return fmt.Sprintf("%s %s(%s)", kind, quoted(p.Provider.Name), fset.Position(p.Provider.Pos))
	case p.Binding != nil:
		return fmt.Sprintf("di.Bind (%s)", fset.Position(p.Binding.Pos))
	case p.Value != nil:
		return fmt.Sprintf("di.Value (%s)", fset.Position(p.Value.Pos))
	case p.Import != nil:
		return fmt.Sprintf("provider set %s(%s)", quoted(p.Import.VarName), fset.Position(p.Import.Pos))
	case p.InjectorArg != nil:
		args := p.InjectorArg.Args
		return fmt.Sprintf("argument %s to injector function %s (%s)", args.Tuple.At(p.InjectorArg.Index).Name(), args.Name, fset.Position(args.Pos))
	case p.Field != nil:
		return fmt.Sprintf("di.FieldsOf (%s)", fset.Position(p.Field.Pos))
	}
	panic("providerSetSrc 没有任何字段被设置")
}

// trace 返回描述 p（可能递归）来源的字符串切片，
// 包含行号。
func (p *providerSetSrc) trace(fset *token.FileSet, typ types.Type) []string {
	var retval []string
	// 只有 Import 需要递归。
	if p.Import != nil {
		if parent := p.Import.srcMap.At(typ); parent != nil {
			retval = append(retval, parent.(*providerSetSrc).trace(fset, typ)...)
		}
	}
	retval = append(retval, p.description(fset, typ))
	return retval
}

// ProviderSet 描述一组 provider。零值表示空集合。
//
type ProviderSet struct {
	// Pos 是创建该集合的 di.NewSet 或 di.Build 调用的位置。
	//
	Pos token.Pos
	// PkgPath 是声明该集合的包的导入路径。
	PkgPath string
	// VarName 是集合的变量名（若来自包级变量）。
	//
	VarName string

	Providers []*Provider
	Bindings  []*IfaceBinding
	Values    []*Value
	Fields    []*Field
	Imports   []*ProviderSet
	// InjectorArgs 仅在 di.Build 时被填充。
	InjectorArgs *InjectorArgs

	// providerMap 将提供的类型映射到 *ProvidedType。
	// 它包含所有被导入的类型。
	providerMap *typeutil.Map

	// srcMap 将提供的类型映射到 *providerSetSrc，用于记录提供该类型的
	// Provider、Binding、Value 或 Import。
	srcMap *typeutil.Map

	// multi 将类型映射到产生它的所有 provider（当该类型存在多个 provider 时）。
	// 用于支持切片（list）注入。
	// 第一个 provider 仍保留在 providerMap 中。
	multi *typeutil.Map // types.Type -> []multiProvider

	// genericProviders 保存可被按需实例化的泛型 provider 函数。
	// 它们通过将请求类型与 provider 输出类型进行统一来匹配。
	//
	genericProviders []*Provider
}

// multiProvider 将 ProvidedType 与其来源配对，用于 list 注入。
type multiProvider struct {
	pt  *ProvidedType
	src *providerSetSrc
}

// Outputs 返回一个新切片，包含该 provider 集合可能产生的类型集合。
// 顺序未定义。
func (set *ProviderSet) Outputs() []types.Type {
	return set.providerMap.Keys()
}

// For 返回给定类型对应的 ProvidedType；若不存在则返回零值。
func (set *ProviderSet) For(t types.Type) ProvidedType {
	pt := set.providerMap.At(t)
	if pt == nil {
		return ProvidedType{}
	}
	return *pt.(*ProvidedType)
}

// allProviders 返回产生 t 的所有 provider，包括主 provider 以及
// 为 list 注入收集的额外 provider。
func (set *ProviderSet) allProviders(t types.Type) []multiProvider {
	if l := set.multi.At(t); l != nil {
		return l.([]multiProvider)
	}
	if pt := set.providerMap.At(t); pt != nil {
		return []multiProvider{{
			pt:  pt.(*ProvidedType),
			src: set.srcMap.At(t).(*providerSetSrc),
		}}
	}
	return nil
}

// IfaceBinding 声明：应使用某个类型来满足给定接口类型的输入。
//
type IfaceBinding struct {
	// Iface 是接口类型，即可以被注入的类型。
	Iface types.Type

	// Provided 始终是可赋值给 Iface 的类型。
	Provided types.Type

	// Pos 是声明该绑定的位置。
	Pos token.Pos
}

// Provider 记录 provider 的签名。provider 是单个 Go 对象，
// 可以是函数或具名结构体类型。
type Provider struct {
	// Pkg 是 Go 对象所在的包。
	Pkg *types.Package

	// Name 是 Go 对象的名称。
	Name string

	// Pos 是定义该 provider 的 func 关键字或类型声明的源码位置。
	//
	Pos token.Pos

	// Args 是该 provider 的数据依赖列表。
	Args []ProviderInput

	// Varargs 表示该 provider 函数是否为可变参数函数。
	Varargs bool

	// IsStruct 表示该 provider 是否为具名结构体类型。
	// 否则为函数。
	IsStruct bool

	// Out 是该 provider 产生的类型集合，至少包含一个类型。
	//
	Out []types.Type

	// HasCleanup 表示该 provider 函数是否返回清理函数。
	// （结构体恒为 false。）
	HasCleanup bool

	// HasErr 表示该 provider 函数是否可能返回错误。
	// （结构体恒为 false。）
	HasErr bool

	// TypeParams 保存泛型 provider 函数的类型参数。
	// 非泛型 provider 为 nil。
	TypeParams *types.TypeParamList

	// TypeArgs 保存用于实例化泛型 provider 的具体类型实参。
	// 非泛型 provider 为 nil。
	TypeArgs []types.Type
}

// ProviderInput 描述 provider 图中的一条入边。
type ProviderInput struct {
	Type types.Type

	// 若 provider 是结构体，FieldName 是要设置的字段名。
	FieldName string
}

// Value 描述一个值表达式。
type Value struct {
	// Pos 是定义该值的表达式的源码位置。
	Pos token.Pos

	// Out 是该值产生的类型。
	Out types.Type

	// expr 是传给 di.Value 的表达式。
	expr ast.Expr

	// info 是该表达式的类型信息。
	info *types.Info
}

// InjectorArg 描述传给注入器函数的某个参数。
type InjectorArg struct {
	// Args 是完整的参数集合。
	Args *InjectorArgs
	// Index 是该参数在 Args.Tuple 中的下标。
	Index int
}

// InjectorArgs 描述传给注入器函数的参数。
type InjectorArgs struct {
	// Name 是注入器函数的名称。
	Name string
	// Tuple 表示参数。
	Tuple *types.Tuple
	// Pos 是注入器函数的源码位置。
	Pos token.Pos
}

// Field 描述从结构体中选出的某个字段。
type Field struct {
	// Parent 是该字段所属的结构体或指向结构体的指针。
	Parent types.Type
	// Name 是字段名。
	Name string
	// Pkg 是结构体所在的包。
	Pkg *types.Package
	// Pos 是字段声明的源码位置。
	//
	Pos token.Pos
	// Out 是该字段提供的类型。第一个元素提供字段类型；
	// 若字段来自指向结构体的指针，则会有第二个元素提供指向字段的指针。
	//
	Out []types.Type
}

// Load 查找所有匹配给定模式的包中的 provider 集合，
// 以及这些 provider 集合的传递依赖。
// 它可能同时返回错误和 Info。模式由底层构建系统定义。
// 对 go 工具而言，见
// https://golang.org/cmd/go/#hdr-Package_lists_and_patterns
//
// wd 是工作目录，env 是加载 pattern 指定的包时使用的
// 环境变量集合。若
// env 为 nil 或空，则视为空变量集合。
// 若存在重复的环境变量，以列表中最后一个为准。
//
func Load(ctx context.Context, wd string, env []string, tags string, patterns []string) (*Info, []error) {
	pkgs, errs := load(ctx, wd, env, tags, patterns)
	if len(errs) > 0 {
		return nil, errs
	}
	if len(pkgs) == 0 {
		return new(Info), nil
	}
	fset := pkgs[0].Fset
	info := &Info{
		Fset: fset,
		Sets: make(map[ProviderSetID]*ProviderSet),
	}
	oc := newObjectCache(pkgs)
	ec := new(errorCollector)
	for _, pkg := range pkgs {
		if isDiImport(pkg.PkgPath) {
			// 标记函数所在的包会干扰分析。
			continue
		}
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			if !isProviderSetType(obj.Type()) {
				continue
			}
			item, errs := oc.get(obj)
			if len(errs) > 0 {
				ec.add(notePositionAll(fset.Position(obj.Pos()), errs)...)
				continue
			}
			pset := item.(*ProviderSet)
			// pset.Name 可能不等于 name，因为它可能是另一个 provider 集合的别名。
			//
			id := ProviderSetID{ImportPath: pset.PkgPath, VarName: name}
			info.Sets[id] = pset
		}
		for _, f := range pkg.Syntax {
			for _, decl := range f.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				buildCall, err := findInjectorBuild(pkg.TypesInfo, fn)
				if err != nil {
					ec.add(notePosition(fset.Position(fn.Pos()), fmt.Errorf("注入 %s：%v", fn.Name.Name, err)))
					continue
				}
				if buildCall == nil {
					continue
				}
				sig := pkg.TypesInfo.ObjectOf(fn.Name).Type().(*types.Signature)
				ins, out, err := injectorFuncSignature(sig)
				if err != nil {
					if w, ok := err.(*diErr); ok {
						ec.add(notePosition(w.position, fmt.Errorf("注入 %s：%v", fn.Name.Name, w.error)))
					} else {
						ec.add(notePosition(fset.Position(fn.Pos()), fmt.Errorf("注入 %s：%v", fn.Name.Name, err)))
					}
					continue
				}
				injectorArgs := &InjectorArgs{
					Name:  fn.Name.Name,
					Tuple: ins,
					Pos:   fn.Pos(),
				}
				set, errs := oc.processNewSet(pkg.TypesInfo, pkg.PkgPath, buildCall, injectorArgs, "")
				if len(errs) > 0 {
					ec.add(notePositionAll(fset.Position(fn.Pos()), errs)...)
					continue
				}
				_, errs = solve(fset, out.out, ins, set)
				if len(errs) > 0 {
					ec.add(mapErrors(errs, func(e error) error {
						if w, ok := e.(*diErr); ok {
							return notePosition(w.position, fmt.Errorf("注入 %s：%v", fn.Name.Name, w.error))
						}
						return notePosition(fset.Position(fn.Pos()), fmt.Errorf("注入 %s：%v", fn.Name.Name, e))
					})...)
					continue
				}
				info.Injectors = append(info.Injectors, &Injector{
					ImportPath: pkg.PkgPath,
					FuncName:   fn.Name.Name,
				})
			}
		}
	}
	return info, ec.errors
}

// load 对匹配给定模式的包做类型检查，
// 并包含所有传递依赖的源码。
// 模式由底层构建系统定义。
// 见 https://golang.org/cmd/go/#hdr-Package_lists_and_patterns
//
// wd 是工作目录，env 是加载 pattern 指定的包时使用的
// 环境变量集合。若
// env 为 nil 或空，则视为空变量集合。
// 若存在重复的环境变量，以列表中最后一个为准。
//
func load(ctx context.Context, wd string, env []string, tags string, patterns []string) ([]*packages.Package, []error) {
	cfg := &packages.Config{
		Context:    ctx,
		Mode:       packages.LoadAllSyntax,
		Dir:        wd,
		Env:        env,
		BuildFlags: []string{"-tags=diinject"},
		// TODO(light): 用 ParseFile 跳过间接包中的函数体和注释。
	}
	if len(tags) > 0 {
		cfg.BuildFlags[0] += " " + tags
	}
	escaped := make([]string, len(patterns))
	for i := range patterns {
		escaped[i] = "pattern=" + patterns[i]
	}
	pkgs, err := packages.Load(cfg, escaped...)
	if err != nil {
		return nil, []error{err}
	}
	var errs []error
	for _, p := range pkgs {
		for _, e := range p.Errors {
			errs = append(errs, e)
		}
	}
	if len(errs) > 0 {
		return nil, errs
	}
	return pkgs, nil
}

// Info 保存 Load 的结果。
type Info struct {
	Fset *token.FileSet

	// Sets 包含初始包中的所有 provider 集合。
	Sets map[ProviderSetID]*ProviderSet

	// Injectors 包含初始包中的所有注入器函数。
	// 顺序未定义。
	Injectors []*Injector
}

// ProviderSetID 标识一个具名 provider 集合。
type ProviderSetID struct {
	ImportPath string
	VarName    string
}

// String 返回形如 ""path/to/pkg".Foo" 的 ID。
func (id ProviderSetID) String() string {
	return strconv.Quote(id.ImportPath) + "." + id.VarName
}

// Injector 描述一个注入器函数。
type Injector struct {
	ImportPath string
	FuncName   string
}

// String 返回形如 ""path/to/pkg".Foo" 的注入器名称。
func (in *Injector) String() string {
	return strconv.Quote(in.ImportPath) + "." + in.FuncName
}

// objectCache 是对象到 di 结构的惰性求值映射。
type objectCache struct {
	fset     *token.FileSet
	packages map[string]*packages.Package
	objects  map[objRef]objCacheEntry
	hasher   typeutil.Hasher
}

type objRef struct {
	importPath string
	name       string
}

type objCacheEntry struct {
	val  interface{} // 可为 *Provider、*ProviderSet、*IfaceBinding 或 *Value
	errs []error
}

func newObjectCache(pkgs []*packages.Package) *objectCache {
	if len(pkgs) == 0 {
		panic("object cache 必须有可用的包")
	}
	oc := &objectCache{
		fset:     pkgs[0].Fset,
		packages: make(map[string]*packages.Package),
		objects:  make(map[objRef]objCacheEntry),
		hasher:   typeutil.MakeHasher(),
	}
	// 深度优先遍历所有依赖，建立导入路径到
	// packages.Package 的映射。go/packages 保证在一次 packages.Load 调用中，
	// 对给定的导入路径 X 只会存在
	// 一个 PkgPath 为 X 的 *packages.Package 值。
	stk := append([]*packages.Package(nil), pkgs...)
	for len(stk) > 0 {
		p := stk[len(stk)-1]
		stk = stk[:len(stk)-1]
		if oc.packages[p.PkgPath] != nil {
			continue
		}
		oc.packages[p.PkgPath] = p
		for _, imp := range p.Imports {
			stk = append(stk, imp)
		}
	}
	return oc
}

// get 将 Go 对象转换为 di 结构。它可能返回 *Provider、
// *IfaceBinding、*ProviderSet、*Value 或 []*Field。
func (oc *objectCache) get(obj types.Object) (val interface{}, errs []error) {
	ref := objRef{
		importPath: obj.Pkg().Path(),
		name:       obj.Name(),
	}
	if ent, cached := oc.objects[ref]; cached {
		return ent.val, append([]error(nil), ent.errs...)
	}
	defer func() {
		oc.objects[ref] = objCacheEntry{
			val:  val,
			errs: append([]error(nil), errs...),
		}
	}()
	switch obj := obj.(type) {
	case *types.Var:
		spec := oc.varDecl(obj)
		if spec == nil || len(spec.Values) == 0 {
			return nil, []error{fmt.Errorf("%v 不是 provider 或 provider 集合", obj)}
		}
		var i int
		for i = range spec.Names {
			if spec.Names[i].Name == obj.Name() {
				break
			}
		}
		pkgPath := obj.Pkg().Path()
		return oc.processExpr(oc.packages[pkgPath].TypesInfo, pkgPath, spec.Values[i], obj.Name())
	case *types.Func:
		return processFuncProvider(oc.fset, obj)
	default:
		return nil, []error{fmt.Errorf("%v 不是 provider 或 provider 集合", obj)}
	}
}

// varDecl 查找定义给定变量的声明。
func (oc *objectCache) varDecl(obj *types.Var) *ast.ValueSpec {
	// TODO(light): 若更高效，可遍历文件建立对象到声明的映射。
	// 参考 https://golang.org/s/types-tutorial
	pkg := oc.packages[obj.Pkg().Path()]
	pos := obj.Pos()
	for _, f := range pkg.Syntax {
		tokenFile := oc.fset.File(f.Pos())
		if base := tokenFile.Base(); base <= int(pos) && int(pos) < base+tokenFile.Size() {
			path, _ := astutil.PathEnclosingInterval(f, pos, pos)
			for _, node := range path {
				if spec, ok := node.(*ast.ValueSpec); ok {
					return spec
				}
			}
		}
	}
	return nil
}

// processExpr 将表达式转换为 di 结构。它可能返回
// *Provider、*IfaceBinding、*ProviderSet、*Value 或 []*Field。
func (oc *objectCache) processExpr(info *types.Info, pkgPath string, expr ast.Expr, varName string) (interface{}, []error) {
	exprPos := oc.fset.Position(expr.Pos())
	expr = astutil.Unparen(expr)
	if obj := qualifiedIdentObject(info, expr); obj != nil {
		item, errs := oc.get(obj)
		return item, mapErrors(errs, func(err error) error {
			return notePosition(exprPos, err)
		})
	}
	if call, ok := expr.(*ast.CallExpr); ok {
		fnObj := qualifiedIdentObject(info, call.Fun)
		if fnObj == nil {
			return nil, []error{notePosition(exprPos, errors.New("未知模式：fnObj 为 nil"))}
		}
		pkg := fnObj.Pkg()
		if pkg == nil {
			return nil, []error{notePosition(exprPos, fmt.Errorf("未知模式：fnObj 中的 pkg 为 nil - %s", fnObj))}
		}
		if !isDiImport(pkg.Path()) {
			return nil, []error{notePosition(exprPos, errors.New("未知模式"))}
		}
		switch fnObj.Name() {
		case "NewSet":
			pset, errs := oc.processNewSet(info, pkgPath, call, nil, varName)
			return pset, notePositionAll(exprPos, errs)
		case "Bind":
			b, err := processBind(oc.fset, info, call)
			if err != nil {
				return nil, []error{notePosition(exprPos, err)}
			}
			return b, nil
		case "Value":
			v, err := processValue(oc.fset, info, call)
			if err != nil {
				return nil, []error{notePosition(exprPos, err)}
			}
			return v, nil
		case "InterfaceValue":
			v, err := processInterfaceValue(oc.fset, info, call)
			if err != nil {
				return nil, []error{notePosition(exprPos, err)}
			}
			return v, nil
		case "Struct":
			s, err := processStructProvider(oc.fset, info, call)
			if err != nil {
				return nil, []error{notePosition(exprPos, err)}
			}
			return s, nil
		case "FieldsOf":
			v, err := processFieldsOf(oc.fset, info, call)
			if err != nil {
				return nil, []error{notePosition(exprPos, err)}
			}
			return v, nil
		default:
			return nil, []error{notePosition(exprPos, errors.New("未知模式"))}
		}
	}
	if tn := structArgType(info, expr); tn != nil {
		p, errs := processStructLiteralProvider(oc.fset, tn)
		if len(errs) > 0 {
			return nil, notePositionAll(exprPos, errs)
		}
		return p, nil
	}
	return nil, []error{notePosition(exprPos, errors.New("未知模式"))}
}

func (oc *objectCache) processNewSet(info *types.Info, pkgPath string, call *ast.CallExpr, args *InjectorArgs, varName string) (*ProviderSet, []error) {
	// 假设 call.Fun 是 di.NewSet 或 di.Build。

	pset := &ProviderSet{
		Pos:          call.Pos(),
		InjectorArgs: args,
		PkgPath:      pkgPath,
		VarName:      varName,
	}
	ec := new(errorCollector)
	for _, arg := range call.Args {
		item, errs := oc.processExpr(info, pkgPath, arg, "")
		if len(errs) > 0 {
			ec.add(errs...)
			continue
		}
		switch item := item.(type) {
		case *Provider:
			pset.Providers = append(pset.Providers, item)
		case *ProviderSet:
			pset.Imports = append(pset.Imports, item)
		case *IfaceBinding:
			pset.Bindings = append(pset.Bindings, item)
		case *Value:
			pset.Values = append(pset.Values, item)
		case []*Field:
			pset.Fields = append(pset.Fields, item...)
		default:
			panic("未知的 item 类型")
		}
	}
	if len(ec.errors) > 0 {
		return nil, ec.errors
	}
	var errs []error
	pset.providerMap, pset.srcMap, errs = buildProviderMap(oc.fset, oc.hasher, pset)
	if len(errs) > 0 {
		return nil, errs
	}
	if errs := verifyAcyclic(pset.providerMap, oc.hasher); len(errs) > 0 {
		return nil, errs
	}
	return pset, nil
}

// structArgType 尝试将表达式解释为简单的结构体类型。
// 它假设所有括号都已被剥离。
func structArgType(info *types.Info, expr ast.Expr) *types.TypeName {
	lit, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil
	}
	tn, ok := qualifiedIdentObject(info, lit.Type).(*types.TypeName)
	if !ok {
		return nil
	}
	if _, isStruct := tn.Type().Underlying().(*types.Struct); !isStruct {
		return nil
	}
	return tn
}

// qualifiedIdentObject 查找标识符或限定标识符对应的对象，
// 若找不到则返回 nil。
// 它还会穿透诸如 Foo[T] 的泛型实例化表达式。
func qualifiedIdentObject(info *types.Info, expr ast.Expr) types.Object {
	switch expr := expr.(type) {
	case *ast.Ident:
		return info.ObjectOf(expr)
	case *ast.SelectorExpr:
		pkgName, ok := expr.X.(*ast.Ident)
		if !ok {
			return nil
		}
		if _, ok := info.ObjectOf(pkgName).(*types.PkgName); !ok {
			return nil
		}
		return info.ObjectOf(expr.Sel)
	case *ast.IndexExpr:
		return qualifiedIdentObject(info, expr.X)
	case *ast.IndexListExpr:
		return qualifiedIdentObject(info, expr.X)
	default:
		return nil
	}
}

// processFuncProvider 为函数声明创建 provider。
func processFuncProvider(fset *token.FileSet, fn *types.Func) (*Provider, []error) {
	sig := fn.Type().(*types.Signature)
	fpos := fn.Pos()
	providerSig, err := funcOutput(sig)
	if err != nil {
		return nil, []error{notePosition(fset.Position(fpos), fmt.Errorf("provider %s 的签名错误：%v", fn.Name(), err))}
	}
	params := sig.Params()
	provider := &Provider{
		Pkg:        fn.Pkg(),
		Name:       fn.Name(),
		Pos:        fn.Pos(),
		Args:       make([]ProviderInput, params.Len()),
		Varargs:    sig.Variadic(),
		Out:        []types.Type{providerSig.out},
		HasCleanup: providerSig.cleanup,
		HasErr:     providerSig.err,
		TypeParams: sig.TypeParams(),
	}
	isGeneric := provider.TypeParams != nil && provider.TypeParams.Len() > 0
	for i := 0; i < params.Len(); i++ {
		provider.Args[i] = ProviderInput{
			Type: params.At(i).Type(),
		}
		if isGeneric {
			continue
		}
		for j := 0; j < i; j++ {
			if types.Identical(provider.Args[i].Type, provider.Args[j].Type) {
				return nil, []error{notePosition(fset.Position(fpos), fmt.Errorf("provider 存在多个类型 %s 的参数", types.TypeString(provider.Args[j].Type, nil)))}
			}
		}
	}
	return provider, nil
}

func injectorFuncSignature(sig *types.Signature) (*types.Tuple, outputSignature, error) {
	out, err := funcOutput(sig)
	if err != nil {
		return nil, outputSignature{}, err
	}
	return sig.Params(), out, nil
}

type outputSignature struct {
	out     types.Type
	cleanup bool
	err     bool
}

// funcOutput 校验注入器或 provider 函数的返回签名。
func funcOutput(sig *types.Signature) (outputSignature, error) {
	results := sig.Results()
	switch results.Len() {
	case 0:
		return outputSignature{}, errors.New("无返回值")
	case 1:
		return outputSignature{out: results.At(0).Type()}, nil
	case 2:
		out := results.At(0).Type()
		switch t := results.At(1).Type(); {
		case types.Identical(t, errorType):
			return outputSignature{out: out, err: true}, nil
		case types.Identical(t, cleanupType):
			return outputSignature{out: out, cleanup: true}, nil
		default:
			return outputSignature{}, fmt.Errorf("第二个返回值类型是 %s；必须是 error 或 func()", types.TypeString(t, nil))
		}
	case 3:
		if t := results.At(1).Type(); !types.Identical(t, cleanupType) {
			return outputSignature{}, fmt.Errorf("第二个返回值类型是 %s；必须是 func()", types.TypeString(t, nil))
		}
		if t := results.At(2).Type(); !types.Identical(t, errorType) {
			return outputSignature{}, fmt.Errorf("第三个返回值类型是 %s；必须是 error", types.TypeString(t, nil))
		}
		return outputSignature{
			out:     results.At(0).Type(),
			cleanup: true,
			err:     true,
		}, nil
	default:
		return outputSignature{}, errors.New("返回值过多")
	}
}

// processStructLiteralProvider 为具名结构体类型创建 provider。
// 它通过 Out 中的两个值产生指针和非指针两种变体。
//
// 这是旧 processStructProvider 的拷贝，现已废弃。
// 它不支持 v0.2 之后引入的任何新特性，请改用新的
// di.Struct 语法。
func processStructLiteralProvider(fset *token.FileSet, typeName *types.TypeName) (*Provider, []error) {
	out := typeName.Type()
	st, ok := out.Underlying().(*types.Struct)
	if !ok {
		return nil, []error{fmt.Errorf("%v 不是结构体", typeName)}
	}

	pos := typeName.Pos()
	fmt.Fprintf(os.Stderr,
		"Warning: %v, see https://godoc.org/github.com/google/wire#Struct for more information.\n",
		notePosition(fset.Position(pos),
			fmt.Errorf("使用结构体字面量注入 %s 已废弃，将在下个版本移除；请改用 di.Struct",
				typeName.Type())))
	provider := &Provider{
		Pkg:      typeName.Pkg(),
		Name:     typeName.Name(),
		Pos:      pos,
		Args:     make([]ProviderInput, st.NumFields()),
		IsStruct: true,
		Out:      []types.Type{out, types.NewPointer(out)},
	}
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		provider.Args[i] = ProviderInput{
			Type:      f.Type(),
			FieldName: f.Name(),
		}
		for j := 0; j < i; j++ {
			if types.Identical(provider.Args[i].Type, provider.Args[j].Type) {
				return nil, []error{notePosition(fset.Position(pos), fmt.Errorf("provider 结构体存在多个类型 %s 的字段", types.TypeString(provider.Args[j].Type, nil)))}
			}
		}
	}
	return provider, nil
}

// processStructProvider 为具名结构体类型创建 provider。
// 它通过 Out 中的两个值产生指针和非指针两种变体。
func processStructProvider(fset *token.FileSet, info *types.Info, call *ast.CallExpr) (*Provider, error) {
	// 假设 call.Fun 是 di.Struct。

	if len(call.Args) < 1 {
		return nil, notePosition(fset.Position(call.Pos()),
			errors.New("Struct 调用必须指定要注入的结构体"))
	}
	const firstArgReqFormat = "Struct 的第一个参数必须是指向具名结构体的指针；实际为 %s"
	structType := info.TypeOf(call.Args[0])
	structPtr, ok := structType.(*types.Pointer)
	if !ok {
		return nil, notePosition(fset.Position(call.Pos()),
			fmt.Errorf(firstArgReqFormat, types.TypeString(structType, nil)))
	}

	st, ok := structPtr.Elem().Underlying().(*types.Struct)
	if !ok {
		return nil, notePosition(fset.Position(call.Pos()),
			fmt.Errorf(firstArgReqFormat, types.TypeString(structPtr, nil)))
	}

	stExpr := call.Args[0].(*ast.CallExpr)
	typeName := qualifiedIdentObject(info, stExpr.Args[0]) // 应为标识符或选择器
	provider := &Provider{
		Pkg:      typeName.Pkg(),
		Name:     typeName.Name(),
		Pos:      typeName.Pos(),
		IsStruct: true,
		Out:      []types.Type{structPtr.Elem(), structPtr},
	}
	if allFields(call) {
		for i := 0; i < st.NumFields(); i++ {
			if isPrevented(st.Tag(i)) {
				continue
			}
			f := st.Field(i)
			provider.Args = append(provider.Args, ProviderInput{
				Type:      f.Type(),
				FieldName: f.Name(),
			})
		}
	} else {
		provider.Args = make([]ProviderInput, len(call.Args)-1)
		for i := 1; i < len(call.Args); i++ {
			v, err := checkField(call.Args[i], st)
			if err != nil {
				return nil, notePosition(fset.Position(call.Pos()), err)
			}
			provider.Args[i-1] = ProviderInput{
				Type:      v.Type(),
				FieldName: v.Name(),
			}
		}
	}
	for i := 0; i < len(provider.Args); i++ {
		for j := 0; j < i; j++ {
			if types.Identical(provider.Args[i].Type, provider.Args[j].Type) {
				f := st.Field(j)
				return nil, notePosition(fset.Position(f.Pos()), fmt.Errorf("provider 结构体存在多个类型 %s 的字段", types.TypeString(provider.Args[j].Type, nil)))
			}
		}
	}
	return provider, nil
}

func allFields(call *ast.CallExpr) bool {
	if len(call.Args) != 2 {
		return false
	}
	b, ok := call.Args[1].(*ast.BasicLit)
	if !ok {
		return false
	}
	return strings.EqualFold(strconv.Quote("*"), b.Value)
}

// isPrevented 判断字段 i 是否被 "-" 标签阻止注入。
// 由于这是 di 唯一使用的标签，可以直接做字符串比较，
// 无需使用 reflect。
func isPrevented(tag string) bool {
	return reflect.StructTag(tag).Get("di") == "-"
}

// processBind 从 di.Bind 调用创建接口绑定。
func processBind(fset *token.FileSet, info *types.Info, call *ast.CallExpr) (*IfaceBinding, error) {
	// 假设 call.Fun 是 di.Bind。

	if len(call.Args) != 2 {
		return nil, notePosition(fset.Position(call.Pos()),
			errors.New("Bind 调用必须恰好接受两个参数"))
	}
	// TODO(light): 验证参数是否为简单表达式。
	ifaceArgType := info.TypeOf(call.Args[0])
	ifacePtr, ok := ifaceArgType.(*types.Pointer)
	if !ok {
		return nil, notePosition(fset.Position(call.Pos()),
			fmt.Errorf("Bind 的第一个参数必须是指向接口类型的指针；实际为 %s", types.TypeString(ifaceArgType, nil)))
	}
	iface := ifacePtr.Elem()
	methodSet, ok := iface.Underlying().(*types.Interface)
	if !ok {
		return nil, notePosition(fset.Position(call.Pos()),
			fmt.Errorf("Bind 的第一个参数必须是指向接口类型的指针；实际为 %s", types.TypeString(ifaceArgType, nil)))
	}

	provided := info.TypeOf(call.Args[1])
	if bindShouldUsePointer(info, call) {
		providedPtr, ok := provided.(*types.Pointer)
		if !ok {
			return nil, notePosition(fset.Position(call.Args[0].Pos()),
				fmt.Errorf("Bind 的第二个参数必须是指针或指向指针的指针；实际为 %s", types.TypeString(provided, nil)))
		}
		provided = providedPtr.Elem()
	}
	if types.Identical(iface, provided) {
		return nil, notePosition(fset.Position(call.Pos()),
			errors.New("不能将接口绑定到自身"))
	}
	if !types.Implements(provided, methodSet) {
		return nil, notePosition(fset.Position(call.Pos()),
			fmt.Errorf("%s 未实现 %s", types.TypeString(provided, nil), types.TypeString(iface, nil)))
	}
	return &IfaceBinding{
		Pos:      call.Pos(),
		Iface:    iface,
		Provided: provided,
	}, nil
}

// processValue 从 di.Value 调用创建值。
func processValue(fset *token.FileSet, info *types.Info, call *ast.CallExpr) (*Value, error) {
	// 假设 call.Fun 是 di.Value。

	if len(call.Args) != 1 {
		return nil, notePosition(fset.Position(call.Pos()), errors.New("Value 调用必须恰好接受一个参数"))
	}
	ok := true
	ast.Inspect(call.Args[0], func(node ast.Node) bool {
		switch expr := node.(type) {
		case nil, *ast.ArrayType, *ast.BasicLit, *ast.BinaryExpr, *ast.ChanType, *ast.CompositeLit, *ast.FuncType, *ast.Ident, *ast.IndexExpr, *ast.InterfaceType, *ast.KeyValueExpr, *ast.MapType, *ast.ParenExpr, *ast.SelectorExpr, *ast.SliceExpr, *ast.StarExpr, *ast.StructType, *ast.TypeAssertExpr:
			// 合法！
		case *ast.UnaryExpr:
			if expr.Op == token.ARROW {
				ok = false
				return false
			}
		case *ast.CallExpr:
			// 仅当是类型转换时才可接受。
			if _, isFunc := info.TypeOf(expr.Fun).(*types.Signature); isFunc {
				ok = false
				return false
			}
		default:
			ok = false
			return false
		}
		return true
	})
	if !ok {
		return nil, notePosition(fset.Position(call.Pos()), errors.New("Value 的参数过于复杂"))
	}
	// 结果类型不能是接口类型；若为接口类型请使用 di.InterfaceValue。
	argType := info.TypeOf(call.Args[0])
	if _, isInterfaceType := argType.Underlying().(*types.Interface); isInterfaceType {
		return nil, notePosition(fset.Position(call.Pos()), fmt.Errorf("Value 的参数不能是接口值（实际为 %s）；请改用 InterfaceValue", types.TypeString(argType, nil)))
	}
	return &Value{
		Pos:  call.Args[0].Pos(),
		Out:  info.TypeOf(call.Args[0]),
		expr: call.Args[0],
		info: info,
	}, nil
}

// processInterfaceValue 从 di.InterfaceValue 调用创建值。
func processInterfaceValue(fset *token.FileSet, info *types.Info, call *ast.CallExpr) (*Value, error) {
	// 假设 call.Fun 是 di.InterfaceValue。

	if len(call.Args) != 2 {
		return nil, notePosition(fset.Position(call.Pos()), errors.New("InterfaceValue 调用必须恰好接受两个参数"))
	}
	ifaceArgType := info.TypeOf(call.Args[0])
	ifacePtr, ok := ifaceArgType.(*types.Pointer)
	if !ok {
		return nil, notePosition(fset.Position(call.Pos()), fmt.Errorf("InterfaceValue 的第一个参数必须是指向接口类型的指针；实际为 %s", types.TypeString(ifaceArgType, nil)))
	}
	iface := ifacePtr.Elem()
	methodSet, ok := iface.Underlying().(*types.Interface)
	if !ok {
		return nil, notePosition(fset.Position(call.Pos()), fmt.Errorf("InterfaceValue 的第一个参数必须是指向接口类型的指针；实际为 %s", types.TypeString(ifaceArgType, nil)))
	}
	provided := info.TypeOf(call.Args[1])
	if !types.Implements(provided, methodSet) {
		return nil, notePosition(fset.Position(call.Pos()), fmt.Errorf("%s 未实现 %s", types.TypeString(provided, nil), types.TypeString(iface, nil)))
	}
	return &Value{
		Pos:  call.Args[1].Pos(),
		Out:  iface,
		expr: call.Args[1],
		info: info,
	}, nil
}

// processFieldsOf 从 di.FieldsOf 调用创建字段切片。
func processFieldsOf(fset *token.FileSet, info *types.Info, call *ast.CallExpr) ([]*Field, error) {
	// 假设 call.Fun 是 di.FieldsOf。

	if len(call.Args) < 2 {
		return nil, notePosition(fset.Position(call.Pos()),
			errors.New("FieldsOf 调用必须指定要提取的字段"))
	}
	const firstArgReqFormat = "FieldsOf 的第一个参数必须是指向结构体的指针或指向指针的指针；实际为 %s"
	structType := info.TypeOf(call.Args[0])
	structPtr, ok := structType.(*types.Pointer)
	if !ok {
		return nil, notePosition(fset.Position(call.Pos()),
			fmt.Errorf(firstArgReqFormat, types.TypeString(structType, nil)))
	}

	var struc *types.Struct
	isPtrToStruct := false
	switch t := structPtr.Elem().Underlying().(type) {
	case *types.Pointer:
		struc, ok = t.Elem().Underlying().(*types.Struct)
		if !ok {
			return nil, notePosition(fset.Position(call.Pos()),
				fmt.Errorf(firstArgReqFormat, types.TypeString(struc, nil)))
		}
		isPtrToStruct = true
	case *types.Struct:
		struc = t
	default:
		return nil, notePosition(fset.Position(call.Pos()),
			fmt.Errorf(firstArgReqFormat, types.TypeString(t, nil)))
	}
	if struc.NumFields() < len(call.Args)-1 {
		return nil, notePosition(fset.Position(call.Pos()),
			fmt.Errorf("字段数量超过了结构体可用的字段数（%d 个字段）", struc.NumFields()))
	}

	fields := make([]*Field, 0, len(call.Args)-1)
	for i := 1; i < len(call.Args); i++ {
		v, err := checkField(call.Args[i], struc)
		if err != nil {
			return nil, notePosition(fset.Position(call.Pos()), err)
		}
		out := []types.Type{v.Type()}
		if isPtrToStruct {
			// 若字段来自指向结构体的指针，
			// 则 di.Fields 还会提供指向该字段的指针。
			out = append(out, types.NewPointer(v.Type()))
		}
		fields = append(fields, &Field{
			Parent: structPtr.Elem(),
			Name:   v.Name(),
			Pkg:    v.Pkg(),
			Pos:    v.Pos(),
			Out:    out,
		})
	}
	return fields, nil
}

// checkField 判断 f 是否为 st 的字段。f 应为包含字段名的字符串。
//
func checkField(f ast.Expr, st *types.Struct) (*types.Var, error) {
	b, ok := f.(*ast.BasicLit)
	if !ok {
		return nil, fmt.Errorf("%v 必须是包含字段名的字符串", f)
	}
	for i := 0; i < st.NumFields(); i++ {
		if strings.EqualFold(strconv.Quote(st.Field(i).Name()), b.Value) {
			if isPrevented(st.Tag(i)) {
				return nil, fmt.Errorf("%s 被 di 阻止注入", b.Value)
			}
			return st.Field(i), nil
		}
	}
	return nil, fmt.Errorf("%s 不是 %s 的字段", b.Value, st.String())
}

// 若 fn 是注入器模板，findInjectorBuild 返回其中的 di.Build 调用。
// 若该函数不是注入器模板则返回 nil。
func findInjectorBuild(info *types.Info, fn *ast.FuncDecl) (*ast.CallExpr, error) {
	if fn.Body == nil {
		return nil, nil
	}
	numStatements := 0
	invalid := false
	var diBuildCall *ast.CallExpr
	for _, stmt := range fn.Body.List {
		switch stmt := stmt.(type) {
		case *ast.ExprStmt:
			numStatements++
			if numStatements > 1 {
				invalid = true
			}
			call, ok := stmt.X.(*ast.CallExpr)
			if !ok {
				continue
			}
			if qualifiedIdentObject(info, call.Fun) == types.Universe.Lookup("panic") {
				if len(call.Args) != 1 {
					continue
				}
				call, ok = call.Args[0].(*ast.CallExpr)
				if !ok {
					continue
				}
			}
			buildObj := qualifiedIdentObject(info, call.Fun)
			if buildObj == nil || buildObj.Pkg() == nil || !isDiImport(buildObj.Pkg().Path()) || buildObj.Name() != "Build" {
				continue
			}
			diBuildCall = call
		case *ast.EmptyStmt:
			// 什么都不做。
		case *ast.ReturnStmt:
			// 允许函数以 return 结尾。
			if numStatements == 0 {
				return nil, nil
			}
		default:
			invalid = true
		}

	}
	if diBuildCall == nil {
		return nil, nil
	}
	if invalid {
		return nil, errors.New("di.Build 调用表明该函数是注入器，但注入器必须只包含 di.Build 调用和一个可选的 return")
	}
	return diBuildCall, nil
}

func isDiImport(path string) bool {
	// TODO(light): 这里依赖当前加载器的实现细节。
	const vendorPart = "vendor/"
	if i := strings.LastIndex(path, vendorPart); i != -1 && (i == 0 || path[i-1] == '/') {
		path = path[i+len(vendorPart):]
	}
	return path == "github.com/jdkhome/gdk/di"
}

func isProviderSetType(t types.Type) bool {
	n, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := n.Obj()
	return obj.Pkg() != nil && isDiImport(obj.Pkg().Path()) && obj.Name() == "ProviderSet"
}

// ProvidedType 表示由某个来源提供的类型。
// 来源可以是 *Provider（provider 函数）、*Value（di.Value）或
// *InjectorArgs（注入器函数的参数）。
// 零值不包含上述任何来源，IsNil 返回 true。
type ProvidedType struct {
	// t 是提供的具体类型。
	t types.Type
	p *Provider
	v *Value
	a *InjectorArg
	f *Field
}

// IsNil 判断 pt 是否为零值。
func (pt ProvidedType) IsNil() bool {
	return pt.p == nil && pt.v == nil && pt.a == nil && pt.f == nil
}

// Type 返回输出类型。
//
//   - 对于函数 provider，是第一个返回值类型。
//   - 对于结构体 provider，是结构体类型或元素为结构体类型的指针类型。
//
//   - 对于值，是表达式的类型。
//   - 对于参数，是参数的类型。
func (pt ProvidedType) Type() types.Type {
	return pt.t
}

// IsProvider 判断 pt 是否指向 Provider。
func (pt ProvidedType) IsProvider() bool {
	return pt.p != nil
}

// IsValue 判断 pt 是否指向 Value。
func (pt ProvidedType) IsValue() bool {
	return pt.v != nil
}

// IsArg 判断 pt 是否指向注入器参数。
func (pt ProvidedType) IsArg() bool {
	return pt.a != nil
}

// IsField 判断 pt 是否指向 Fields。
func (pt ProvidedType) IsField() bool {
	return pt.f != nil
}

// Provider 将 pt 作为 Provider 指针返回；若 pt 不指向 Provider 则 panic。
//
func (pt ProvidedType) Provider() *Provider {
	if pt.p == nil {
		panic("ProvidedType 不包含 Provider")
	}
	return pt.p
}

// Value 将 pt 作为 Value 指针返回；若 pt 不指向 Value 则 panic。
//
func (pt ProvidedType) Value() *Value {
	if pt.v == nil {
		panic("ProvidedType 不包含 Value")
	}
	return pt.v
}

// Arg 将 pt 作为表示注入器参数的 *InjectorArg 返回；
// 若 pt 不指向参数则 panic。
func (pt ProvidedType) Arg() *InjectorArg {
	if pt.a == nil {
		panic("ProvidedType 不包含 Arg")
	}
	return pt.a
}

// Field 将 pt 作为 Field 指针返回；若 pt 不指向结构体字段则 panic。
//
func (pt ProvidedType) Field() *Field {
	if pt.f == nil {
		panic("ProvidedType 不包含 Field")
	}
	return pt.f
}

// bindShouldUsePointer 加载用户在注入器中导入的 di 包，
// 该调用是 di 标记函数调用。
func bindShouldUsePointer(info *types.Info, call *ast.CallExpr) bool {
	// 这些类型断言不应失败，否则会 panic。
	fun := call.Fun.(*ast.SelectorExpr)                 // di.Bind
	pkgName := fun.X.(*ast.Ident)                       // di
	diName := info.ObjectOf(pkgName).(*types.PkgName) // di 包
	return diName.Imported().Scope().Lookup("bindToUsePointer") != nil
}
