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
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strings"

	"golang.org/x/tools/go/types/typeutil"
)

type callKind int

const (
	funcProviderCall callKind = iota
	structProvider
	valueExpr
	selectorExpr
	sliceExpr
)

// call 表示注入器函数的一个步骤。根据 kind 的值，
// 它可以是函数调用或复合结构体字面量。

type call struct {
	// kind 表示要使用的代码模式。
	kind callKind

	// out 是该步骤产生的类型。
	out types.Type

	// pkg 和 name 标识以下之一：
	// 1) kind == funcProviderCall 时要调用的 provider；
	// 2) kind == structProvider 时要构造的类型；
	// 3) kind == selectorExpr 时要选择的名称。
	pkg  *types.Package
	name string

	// args 是调用 provider 时使用的参数列表。每个元素为：
	// a) 输入之一（args[i] < len(given)）；
	// b) 之前某次 provider 调用的结果（args[i] >= len(given)）。
	//
	// kind == valueExpr 时为 nil。
	//
	// 若 kind == selectorExpr，则切片长度为 1，
	// "参数" 表示要访问字段的值。
	args []int

	// varargs 表示 provider 函数是否为可变参数函数。
	varargs bool

	// typeArgs 保存泛型 provider 调用的类型实参。
	// 非泛型 provider 为 nil。
	typeArgs []types.Type

	// fieldNames 将参数映射到结构体字段名。
	// 仅当 kind == structProvider 时设置。
	fieldNames []string

	// ins 是该调用作为参数接收的类型列表。
	// kind == valueExpr 时为 nil。
	ins []types.Type

	// 以下字段仅当 kind == funcProviderCall 时设置：

	// hasCleanup 表示 provider 调用是否返回清理函数。
	hasCleanup bool
	// hasErr 表示 provider 调用是否返回错误。
	hasErr bool

	// 以下字段仅当 kind == valueExpr 时设置：

	valueExpr     ast.Expr
	valueTypeInfo *types.Info

	// 以下字段仅当 kind == selectorExpr 时设置：

	ptrToField bool
}

// solve 查找产生输出类型所需的调用序列，
// 可带一组可选的输入。
func solve(fset *token.FileSet, out types.Type, given *types.Tuple, set *ProviderSet) ([]call, []error) {
	ec := new(errorCollector)

	// 开始建立类型到给定类型局部变量的映射。
	// 前 len(given) 个局部变量是给定类型。
	index := new(typeutil.Map)
	for i := 0; i < given.Len(); i++ {
		index.Set(given.At(i).Type(), i)
	}

	errAbort := errors.New("访问失败")
	var used []*providerSetSrc
	var calls []call

	noProviderError := func(t types.Type, path []types.Type) error {
		sb := new(strings.Builder)
		if len(path) == 0 {
			fmt.Fprintf(sb, "未找到类型 %s 的 provider（注入器输出）", types.TypeString(t, nil))
		} else {
			fmt.Fprintf(sb, "未找到类型 %s 的 provider", types.TypeString(t, nil))
			for i := len(path) - 1; i >= 0; i-- {
				fmt.Fprintf(sb, "\n被 %s 需要（位于 %s）", types.TypeString(path[i], nil), set.srcMap.At(path[i]).(*providerSetSrc).description(fset, path[i]))
			}
		}
		return errors.New(sb.String())
	}

	// solveProvided 解析产生 want 的某个特定 provider（pt），
	// 追加必要的调用并返回该值的下标。
	var solveProvided func(pt *ProvidedType, want types.Type, path []types.Type) (int, bool)
	// solveType 解析单个类型，返回其值下标。
	var solveType func(t types.Type, path []types.Type) (int, bool)
	// solveList 通过收集 elem 的所有 provider 来解析切片类型。
	var solveList func(elem types.Type, path []types.Type) (int, bool)
	// solveGeneric 通过实例化泛型 provider 来解析类型。
	var solveGeneric func(t types.Type, path []types.Type) (int, bool)

	solveProvided = func(pt *ProvidedType, want types.Type, path []types.Type) (int, bool) {
		// 接口绑定不产生调用；解析具体类型。
		if concrete := pt.Type(); !types.Identical(concrete, want) {
			return solveType(concrete, append(path, want))
		}

		switch {
		case pt.IsArg():
			if i := index.At(want); i != nil {
				return i.(int), true
			}
			index.Set(want, errAbort)
			return 0, false
		case pt.IsProvider():
			p := pt.Provider()
			args := make([]int, len(p.Args))
			ins := make([]types.Type, len(p.Args))
			for i := range p.Args {
				a := p.Args[i]
				v, ok := solveType(a.Type, append(path, want))
				if !ok {
					index.Set(want, errAbort)
					return 0, false
				}
				args[i] = v
				ins[i] = a.Type
			}
			idx := given.Len() + len(calls)
			kind := funcProviderCall
			fieldNames := []string(nil)
			if p.IsStruct {
				kind = structProvider
				for _, arg := range p.Args {
					fieldNames = append(fieldNames, arg.FieldName)
				}
			}
			calls = append(calls, call{
				kind:       kind,
				pkg:        p.Pkg,
				name:       p.Name,
				args:       args,
				varargs:    p.Varargs,
				typeArgs:   p.TypeArgs,
				fieldNames: fieldNames,
				ins:        ins,
				out:        want,
				hasCleanup: p.HasCleanup,
				hasErr:     p.HasErr,
			})
			index.Set(want, idx)
			return idx, true
		case pt.IsValue():
			v := pt.Value()
			idx := given.Len() + len(calls)
			calls = append(calls, call{
				kind:          valueExpr,
				out:           want,
				valueExpr:     v.expr,
				valueTypeInfo: v.info,
			})
			index.Set(want, idx)
			return idx, true
		case pt.IsField():
			f := pt.Field()
			pv, ok := solveType(f.Parent, append(path, want))
			if !ok {
				index.Set(want, errAbort)
				return 0, false
			}
			idx := given.Len() + len(calls)
			ptrToField := len(f.Out) == 2 && types.Identical(want, f.Out[1])
			calls = append(calls, call{
				kind:       selectorExpr,
				pkg:        f.Pkg,
				name:       f.Name,
				out:        want,
				args:       []int{pv},
				ptrToField: ptrToField,
			})
			index.Set(want, idx)
			return idx, true
		default:
			panic("ProviderSet.For 返回了未知的值")
		}
	}

	solveList = func(elem types.Type, path []types.Type) (int, bool) {
		providers := set.allProviders(elem)
		if len(providers) == 0 {
			ec.add(noProviderError(elem, path))
			return 0, false
		}
		args := make([]int, 0, len(providers))
		for _, mp := range providers {
			used = append(used, mp.src)
			idx, ok := solveProvided(mp.pt, elem, path)
			if !ok {
				return 0, false
			}
			args = append(args, idx)
		}
		sliceType := types.NewSlice(elem)
		idx := given.Len() + len(calls)
		calls = append(calls, call{
			kind: sliceExpr,
			out:  sliceType,
			args: args,
		})
		index.Set(sliceType, idx)
		return idx, true
	}

	solveGeneric = func(t types.Type, path []types.Type) (int, bool) {
		for _, gp := range set.genericProviders {
			bindings := make(map[*types.TypeParam]types.Type)
			if !unify(t, gp.Out[0], bindings) {
				continue
			}

			concrete := &Provider{
				Pkg:        gp.Pkg,
				Name:       gp.Name,
				Pos:        gp.Pos,
				Varargs:    gp.Varargs,
				IsStruct:   gp.IsStruct,
				HasCleanup: gp.HasCleanup,
				HasErr:     gp.HasErr,
			}
			typeArgs := make([]types.Type, gp.TypeParams.Len())
			for i := 0; i < gp.TypeParams.Len(); i++ {
				if v, ok := bindings[gp.TypeParams.At(i)]; ok {
					typeArgs[i] = v
				}
			}
			concrete.TypeArgs = typeArgs
			for _, a := range gp.Args {
				concrete.Args = append(concrete.Args, ProviderInput{
					Type: substType(a.Type, bindings),
				})
			}
			concrete.Out = []types.Type{substType(gp.Out[0], bindings)}

			used = append(used, &providerSetSrc{Provider: gp})
			return solveProvided(&ProvidedType{t: t, p: concrete}, t, path)
		}
		return 0, false
	}

	solveType = func(t types.Type, path []types.Type) (int, bool) {
		if i := index.At(t); i != nil {
			if i == errAbort {
				return 0, false
			}
			return i.(int), true
		}

		pv := set.For(t)
		if pv.IsNil() {
			// 切片注入：收集元素类型的所有 provider。
			if slice, ok := t.(*types.Slice); ok {
				return solveList(slice.Elem(), path)
			}
			// 泛型 provider 实例化。
			if idx, ok := solveGeneric(t, path); ok {
				return idx, true
			}
			ec.add(noProviderError(t, path))
			index.Set(t, errAbort)
			return 0, false
		}

		src := set.srcMap.At(t).(*providerSetSrc)
		used = append(used, src)

		// 单值注入存在多个 provider 的类型是有歧义的。
		if set.multi != nil && set.multi.At(t) != nil {
			ec.add(fmt.Errorf("类型 %s 存在多个 provider；请使用 []%s 注入全部", types.TypeString(t, nil), types.TypeString(t, nil)))
			index.Set(t, errAbort)
			return 0, false
		}

		return solveProvided(&pv, t, path)
	}

	if _, ok := solveType(out, nil); !ok {
		return nil, ec.errors
	}
	if len(ec.errors) > 0 {
		return nil, ec.errors
	}
	if errs := verifyArgsUsed(set, used); len(errs) > 0 {
		return nil, errs
	}
	return calls, nil
}

// unify 将 target 与 pattern 匹配，把 pattern 中出现的类型参数
// 绑定到具体类型。它判断是否可以通过给每个类型参数赋予一致的具体类型，
// 使 target 与 pattern 变得相同。
func unify(target, pattern types.Type, bindings map[*types.TypeParam]types.Type) bool {
	if tp, ok := pattern.(*types.TypeParam); ok {
		if existing, ok := bindings[tp]; ok {
			return types.Identical(existing, target)
		}
		bindings[tp] = target
		return true
	}

	switch p := pattern.(type) {
	case *types.Pointer:
		t, ok := target.(*types.Pointer)
		if !ok {
			return false
		}
		return unify(t.Elem(), p.Elem(), bindings)
	case *types.Slice:
		t, ok := target.(*types.Slice)
		if !ok {
			return false
		}
		return unify(t.Elem(), p.Elem(), bindings)
	case *types.Array:
		t, ok := target.(*types.Array)
		if !ok || t.Len() != p.Len() {
			return false
		}
		return unify(t.Elem(), p.Elem(), bindings)
	case *types.Map:
		t, ok := target.(*types.Map)
		if !ok {
			return false
		}
		return unify(t.Key(), p.Key(), bindings) && unify(t.Elem(), p.Elem(), bindings)
	case *types.Chan:
		t, ok := target.(*types.Chan)
		if !ok || t.Dir() != p.Dir() {
			return false
		}
		return unify(t.Elem(), p.Elem(), bindings)
	case *types.Named:
		t, ok := target.(*types.Named)
		if !ok || t.Obj() != p.Obj() {
			return false
		}
		ta, pa := t.TypeArgs(), p.TypeArgs()
		if ta.Len() != pa.Len() {
			return false
		}
		for i := 0; i < pa.Len(); i++ {
			if !unify(ta.At(i), pa.At(i), bindings) {
				return false
			}
		}
		return true
	default:
		return types.Identical(target, pattern)
	}
}

// substType 将 t 中的类型参数替换为它们的具体绑定。
func substType(t types.Type, bindings map[*types.TypeParam]types.Type) types.Type {
	switch t := t.(type) {
	case *types.TypeParam:
		if v, ok := bindings[t]; ok {
			return v
		}
		return t
	case *types.Pointer:
		return types.NewPointer(substType(t.Elem(), bindings))
	case *types.Slice:
		return types.NewSlice(substType(t.Elem(), bindings))
	case *types.Array:
		return types.NewArray(substType(t.Elem(), bindings), t.Len())
	case *types.Map:
		return types.NewMap(substType(t.Key(), bindings), substType(t.Elem(), bindings))
	case *types.Chan:
		return types.NewChan(t.Dir(), substType(t.Elem(), bindings))
	case *types.Named:
		if t.TypeArgs().Len() == 0 {
			return t
		}
		args := make([]types.Type, t.TypeArgs().Len())
		changed := false
		for i := 0; i < t.TypeArgs().Len(); i++ {
			args[i] = substType(t.TypeArgs().At(i), bindings)
			if args[i] != t.TypeArgs().At(i) {
				changed = true
			}
		}
		if !changed {
			return t
		}
		inst, err := types.Instantiate(nil, t.Origin(), args, false)
		if err != nil {
			return t
		}
		return inst
	default:
		return t
	}
}

// verifyArgsUsed 确保 set 中的所有参数都在 solve 过程中被使用。
func verifyArgsUsed(set *ProviderSet, used []*providerSetSrc) []error {
	var errs []error
	for _, imp := range set.Imports {
		found := false
		for _, u := range used {
			if u.Import == imp {
				found = true
				break
			}
		}
		if !found {
			if imp.VarName == "" {
				errs = append(errs, errors.New("未使用的 provider 集合"))
			} else {
				errs = append(errs, fmt.Errorf("未使用的 provider 集合 %q", imp.VarName))
			}
		}
	}
	for _, p := range set.Providers {
		// 泛型 provider 是模板；跳过对它们的未使用检查。
		if p.TypeParams != nil && p.TypeParams.Len() > 0 {
			continue
		}
		found := false
		for _, u := range used {
			if u.Provider == p {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, fmt.Errorf("未使用的 provider %q", p.Pkg.Name()+"."+p.Name))
		}
	}
	for _, v := range set.Values {
		found := false
		for _, u := range used {
			if u.Value == v {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, fmt.Errorf("未使用的 %s 类型值", types.TypeString(v.Out, nil)))
		}
	}
	for _, b := range set.Bindings {
		found := false
		for _, u := range used {
			if u.Binding == b {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, fmt.Errorf("未使用的到 %s 类型的接口绑定", types.TypeString(b.Iface, nil)))
		}
	}
	for _, f := range set.Fields {
		found := false
		for _, u := range used {
			if u.Field == f {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, fmt.Errorf("未使用的字段 %q.%s", f.Parent, f.Name))
		}
	}
	return errs
}

// buildProviderMap 为给定 provider 集合创建 providerMap 和 srcMap 字段。
// 给定 provider 集合的 providerMap 和 srcMap 字段
// 会被忽略。
func buildProviderMap(fset *token.FileSet, hasher typeutil.Hasher, set *ProviderSet) (*typeutil.Map, *typeutil.Map, []error) {
	providerMap := new(typeutil.Map)
	providerMap.SetHasher(hasher)
	srcMap := new(typeutil.Map) // 指向 *providerSetSrc
	srcMap.SetHasher(hasher)
	multi := new(typeutil.Map) // 指向 []multiProvider
	multi.SetHasher(hasher)
	set.multi = multi

	// recordProvider 为 typ 存储 provider。第一个 provider 存入 providerMap；
	// 后续 provider 被收集到 multi 中用于 list 注入，
	// 而不是报错。
	recordProvider := func(typ types.Type, pt *ProvidedType, src *providerSetSrc) {
		prevSrc := srcMap.At(typ)
		if prevSrc == nil {
			providerMap.Set(typ, pt)
			srcMap.Set(typ, src)
			return
		}
		var list []multiProvider
		if l := multi.At(typ); l != nil {
			list = l.([]multiProvider)
		} else {
			list = []multiProvider{{
				pt:  providerMap.At(typ).(*ProvidedType),
				src: prevSrc.(*providerSetSrc),
			}}
		}
		list = append(list, multiProvider{pt: pt, src: src})
		multi.Set(typ, list)
	}

	ec := new(errorCollector)
	// 处理注入器参数。重复的参数类型始终是错误。
	if set.InjectorArgs != nil {
		givens := set.InjectorArgs.Tuple
		for i := 0; i < givens.Len(); i++ {
			typ := givens.At(i).Type()
			arg := &InjectorArg{Args: set.InjectorArgs, Index: i}
			src := &providerSetSrc{InjectorArg: arg}
			if prevSrc := srcMap.At(typ); prevSrc != nil {
				ec.add(bindingConflictError(fset, typ, set, src, prevSrc.(*providerSetSrc)))
				continue
			}
			providerMap.Set(typ, &ProvidedType{t: typ, a: arg})
			srcMap.Set(typ, src)
		}
	}
	// 处理导入。冲突会被收集用于 list 注入。
	for _, imp := range set.Imports {
		src := &providerSetSrc{Import: imp}
		imp.providerMap.Iterate(func(k types.Type, _ interface{}) {
			for _, mp := range imp.allProviders(k) {
				recordProvider(k, mp.pt, src)
			}
		})
		set.genericProviders = append(set.genericProviders, imp.genericProviders...)
	}
	if len(ec.errors) > 0 {
		return nil, nil, ec.errors
	}

	// 处理新集合中的非绑定 provider。
	for _, p := range set.Providers {
		if p.TypeParams != nil && p.TypeParams.Len() > 0 {
			set.genericProviders = append(set.genericProviders, p)
			continue
		}
		src := &providerSetSrc{Provider: p}
		for _, typ := range p.Out {
			recordProvider(typ, &ProvidedType{t: typ, p: p}, src)
		}
	}
	for _, v := range set.Values {
		src := &providerSetSrc{Value: v}
		recordProvider(v.Out, &ProvidedType{t: v.Out, v: v}, src)
	}
	for _, f := range set.Fields {
		src := &providerSetSrc{Field: f}
		for _, typ := range f.Out {
			recordProvider(typ, &ProvidedType{t: typ, f: f}, src)
		}
	}
	if len(ec.errors) > 0 {
		return nil, nil, ec.errors
	}

	// 处理集合中的绑定。必须在其他 provider 之后处理，
	// 以确保具体类型已被提供。
	for _, b := range set.Bindings {
		src := &providerSetSrc{Binding: b}
		concrete := providerMap.At(b.Provided)
		if concrete == nil {
			setName := set.VarName
			if setName == "" {
				setName = "provider set"
			}
			ec.add(notePosition(fset.Position(b.Pos), fmt.Errorf("di.Bind 将具体类型 %q 绑定到接口 %q，但 %s 不包含类型 %q 的 provider", b.Provided, b.Iface, setName, b.Provided)))
			continue
		}
		recordProvider(b.Iface, concrete.(*ProvidedType), src)
	}
	if len(ec.errors) > 0 {
		return nil, nil, ec.errors
	}
	return providerMap, srcMap, nil
}

func verifyAcyclic(providerMap *typeutil.Map, hasher typeutil.Hasher) []error {
	// 我们必须访问 provider map 中的每个 provider 类型，但
	// 没有明确的起点，且可能存在多个
	// 不同的图。因此，我们从每个 provider 开始深度优先搜索，
	// 但保留已访问 provider 的共享记录以避免
	// 重复工作。
	visited := new(typeutil.Map) // 指向 bool
	visited.SetHasher(hasher)
	ec := new(errorCollector)
	// 对输出类型排序，使循环相关的错误保持一致。
	outputs := providerMap.Keys()
	sort.Slice(outputs, func(i, j int) bool { return types.TypeString(outputs[i], nil) < types.TypeString(outputs[j], nil) })
	for _, root := range outputs {
		// 使用遍历 provider map 的路径栈进行深度优先搜索。
		stk := [][]types.Type{{root}}
		for len(stk) > 0 {
			curr := stk[len(stk)-1]
			stk = stk[:len(stk)-1]
			head := curr[len(curr)-1]
			if v, _ := visited.At(head).(bool); v {
				continue
			}
			visited.Set(head, true)
			x := providerMap.At(head)
			if x == nil {
				// 叶子：输入。
				continue
			}
			pt := x.(*ProvidedType)
			switch {
			case pt.IsValue():
				// 叶子：值没有依赖。
			case pt.IsArg():
				// 注入器参数没有依赖。
			case pt.IsProvider() || pt.IsField():
				var args []types.Type
				if pt.IsProvider() {
					for _, arg := range pt.Provider().Args {
						args = append(args, arg.Type)
					}
				} else {
					args = append(args, pt.Field().Parent)
				}
				for _, a := range args {
					hasCycle := false
					for i, b := range curr {
						if types.Identical(a, b) {
							sb := new(strings.Builder)
							fmt.Fprintf(sb, "类型 %s 存在循环依赖：\n", types.TypeString(a, nil))
							for j := i; j < len(curr); j++ {
								t := providerMap.At(curr[j]).(*ProvidedType)
								if t.IsProvider() {
									p := t.Provider()
									fmt.Fprintf(sb, "%s (%s.%s) ->\n", types.TypeString(curr[j], nil), p.Pkg.Path(), p.Name)
								} else {
									p := t.Field()
									fmt.Fprintf(sb, "%s (%s.%s) ->\n", types.TypeString(curr[j], nil), p.Parent, p.Name)
								}
							}
							fmt.Fprintf(sb, "%s", types.TypeString(a, nil))
							ec.add(errors.New(sb.String()))
							hasCycle = true
							break
						}
					}
					if !hasCycle {
						next := append(append([]types.Type(nil), curr...), a)
						stk = append(stk, next)
					}
				}
			default:
				panic("无效的 provider map 值")
			}
		}
	}
	return ec.errors
}

// bindingConflictError 创建一个新错误，描述同一输出类型
// 存在多个绑定的情况。
func bindingConflictError(fset *token.FileSet, typ types.Type, set *ProviderSet, cur, prev *providerSetSrc) error {
	sb := new(strings.Builder)
	if set.VarName != "" {
		fmt.Fprintf(sb, "%s has ", set.VarName)
	}
	fmt.Fprintf(sb, "类型 %s 存在多个绑定\n", types.TypeString(typ, nil))
	fmt.Fprintf(sb, "当前：\n<- %s\n", strings.Join(cur.trace(fset, typ), "\n<- "))
	fmt.Fprintf(sb, "之前：\n<- %s", strings.Join(prev.trace(fset, typ), "\n<- "))
	return notePosition(fset.Position(set.Pos), errors.New(sb.String()))
}
