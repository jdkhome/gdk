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

// Package di 以 Go 库的形式提供编译期依赖注入逻辑。

package di

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/printer"
	"go/token"
	"go/types"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

// GenerateResult 保存某次 Generate 调用针对一个包的结果。
type GenerateResult struct {
	// PkgPath 是包的 PkgPath。
	PkgPath string
	// OutputPath 是生成结果的写入路径。
	// 若有错误则可能为空。
	OutputPath string
	// Content 是生成后已 gofmt 的源码。若生成过程中出错则可能为 nil。
	Content []byte
	// Errs 是生成过程中发现的错误切片。
	Errs []error
}

// Commit 将生成的文件写入磁盘。
func (gen GenerateResult) Commit() error {
	if len(gen.Content) == 0 {
		return nil
	}
	return ioutil.WriteFile(gen.OutputPath, gen.Content, 0666)
}

// GenerateOptions 保存 Generate 的选项。
type GenerateOptions struct {
	// Header 将被插入到每个生成文件的开头。
	Header           []byte
	PrefixOutputFile string
	Tags             string
}

// Generate 对匹配给定模式的包执行依赖注入，并为每个包返回一个
// GenerateResult。包模式由底层构建系统定义；对 go 工具而言，见
// https://golang.org/cmd/go/#hdr-Package_lists_and_patterns
//
// wd 是工作目录，env 是加载 pkgPattern 指定的包时使用的环境变量集合。
// 若 env 为 nil 或空，则视为空变量集合。若存在重复的环境变量，以列表中
// 最后一个为准。
//
// 若加载包失败，Generate 可能返回一个或多个错误。
func Generate(ctx context.Context, wd string, env []string, patterns []string, opts *GenerateOptions) ([]GenerateResult, []error) {
	if opts == nil {
		opts = &GenerateOptions{}
	}
	pkgs, errs := load(ctx, wd, env, opts.Tags, patterns)
	if len(errs) > 0 {
		return nil, errs
	}
	generated := make([]GenerateResult, len(pkgs))
	for i, pkg := range pkgs {
		generated[i].PkgPath = pkg.PkgPath
		outDir, err := detectOutputDir(pkg.GoFiles)
		if err != nil {
			generated[i].Errs = append(generated[i].Errs, err)
			continue
		}
		generated[i].OutputPath = filepath.Join(outDir, opts.PrefixOutputFile+"di_gen.go")
		g := newGen(pkg)
		injectorFiles, errs := generateInjectors(g, pkg)
		if len(errs) > 0 {
			generated[i].Errs = errs
			continue
		}
		copyNonInjectorDecls(g, injectorFiles, pkg.TypesInfo)
		goSrc := g.frame(opts.Tags)
		if len(opts.Header) > 0 {
			goSrc = append(opts.Header, goSrc...)
		}
		fmtSrc, err := format.Source(goSrc)
		if err != nil {
			// 这很可能是生成源码质量不佳导致的 bug。
			// 记录错误，同时保留未格式化的源码。
			generated[i].Errs = append(generated[i].Errs, err)
		} else {
			goSrc = fmtSrc
		}
		generated[i].Content = goSrc
	}
	return generated, nil
}

func detectOutputDir(paths []string) (string, error) {
	if len(paths) == 0 {
		return "", errors.New("没有可用于推导输出目录的文件")
	}
	dir := filepath.Dir(paths[0])
	for _, p := range paths[1:] {
		if dir2 := filepath.Dir(p); dir2 != dir {
			return "", fmt.Errorf("发现冲突的目录 %q 和 %q", dir, dir2)
		}
	}
	return dir, nil
}

// generateInjectors 为给定包生成注入器。
func generateInjectors(g *gen, pkg *packages.Package) (injectorFiles []*ast.File, _ []error) {
	oc := newObjectCache([]*packages.Package{pkg})
	injectorFiles = make([]*ast.File, 0, len(pkg.Syntax))
	ec := new(errorCollector)
	for _, f := range pkg.Syntax {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			buildCall, err := findInjectorBuild(pkg.TypesInfo, fn)
			if err != nil {
				ec.add(err)
				continue
			}
			if buildCall == nil {
				continue
			}
			if len(injectorFiles) == 0 || injectorFiles[len(injectorFiles)-1] != f {
				// 这是该文件生成的第一个注入器。
				// 写入文件头。
				name := filepath.Base(g.pkg.Fset.File(f.Pos()).Name())
				g.p("// 来自 %s 的注入器：\n\n", name)
				injectorFiles = append(injectorFiles, f)
			}
			sig := pkg.TypesInfo.ObjectOf(fn.Name).Type().(*types.Signature)
			ins, _, err := injectorFuncSignature(sig)
			if err != nil {
				if w, ok := err.(*diErr); ok {
					ec.add(notePosition(w.position, fmt.Errorf("注入 %s：%v", fn.Name.Name, w.error)))
				} else {
					ec.add(notePosition(g.pkg.Fset.Position(fn.Pos()), fmt.Errorf("注入 %s：%v", fn.Name.Name, err)))
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
				ec.add(notePositionAll(g.pkg.Fset.Position(fn.Pos()), errs)...)
				continue
			}
			if errs := g.inject(fn.Pos(), fn.Name.Name, sig, set, fn.Doc); len(errs) > 0 {
				ec.add(errs...)
				continue
			}
		}

		for _, impt := range f.Imports {
			if impt.Name != nil && impt.Name.Name == "_" {
				g.anonImports[impt.Path.Value] = true
			}
		}
	}
	if len(ec.errors) > 0 {
		return nil, ec.errors
	}
	return injectorFiles, nil
}

// copyNonInjectorDecls 将给定文件中的非注入器声明
// 复制到生成结果中。
func copyNonInjectorDecls(g *gen, files []*ast.File, info *types.Info) {
	for _, f := range files {
		name := filepath.Base(g.pkg.Fset.File(f.Pos()).Name())
		first := true
		for _, decl := range f.Decls {
			switch decl := decl.(type) {
			case *ast.FuncDecl:
				// 可以忽略错误，因为所有错误情况都应已被
				// 过滤掉。
				if buildCall, _ := findInjectorBuild(info, decl); buildCall != nil {
					continue
				}
			case *ast.GenDecl:
				if decl.Tok == token.IMPORT {
					continue
				}
			default:
				continue
			}
			if first {
				g.p("// %s:\n\n", name)
				first = false
			}
			// TODO(light): 在每个声明顶部添加行号。
			g.writeAST(info, decl)
			g.p("\n\n")
		}
	}
}

// importInfo 保存导入相关的信息。
type importInfo struct {
	// name 是生成源码中使用的标识符。
	name string
	// 若导入被赋予与包标识符不匹配的标识符，differs 为 true。
	differs bool
}

// gen 是文件级生成器状态。
type gen struct {
	pkg         *packages.Package
	buf         bytes.Buffer
	imports     map[string]importInfo
	anonImports map[string]bool
	values      map[ast.Expr]string
}

func newGen(pkg *packages.Package) *gen {
	return &gen{
		pkg:         pkg,
		anonImports: make(map[string]bool),
		imports:     make(map[string]importInfo),
		values:      make(map[ast.Expr]string),
	}
}

// frame 将已构建的源码主体封装为未格式化的 Go 源文件。
func (g *gen) frame(tags string) []byte {
	if g.buf.Len() == 0 {
		return nil
	}
	var buf bytes.Buffer
	if len(tags) > 0 {
		tags = fmt.Sprintf(" gen -tags \"%s\"", tags)
	}
	buf.WriteString("// Code generated by di. DO NOT EDIT.\n\n")
	buf.WriteString("//go:generate go run -mod=mod github.com/jdkhome/gdk/di/cmd/di" + tags + "\n")
	buf.WriteString("//+build !diinject\n\n")
	buf.WriteString("package ")
	buf.WriteString(g.pkg.Name)
	buf.WriteString("\n\n")
	if len(g.imports) > 0 {
		buf.WriteString("import (\n")
		imps := make([]string, 0, len(g.imports))
		for path := range g.imports {
			imps = append(imps, path)
		}
		sort.Strings(imps)
		for _, path := range imps {
			// 若与包名一致，则省略本地包标识符。
			info := g.imports[path]
			if info.differs {
				fmt.Fprintf(&buf, "\t%s %q\n", info.name, path)
			} else {
				fmt.Fprintf(&buf, "\t%q\n", path)
			}
		}
		buf.WriteString(")\n\n")
	}
	if len(g.anonImports) > 0 {
		buf.WriteString("import (\n")
		anonImps := make([]string, 0, len(g.anonImports))
		for path := range g.anonImports {
			anonImps = append(anonImps, path)
		}
		sort.Strings(anonImps)

		for _, path := range anonImps {
			fmt.Fprintf(&buf, "\t_ %s\n", path)
		}
		buf.WriteString(")\n\n")
	}
	buf.Write(g.buf.Bytes())
	return buf.Bytes()
}

// inject 为注入器生成代码。
func (g *gen) inject(pos token.Pos, name string, sig *types.Signature, set *ProviderSet, doc *ast.CommentGroup) []error {
	injectSig, err := funcOutput(sig)
	if err != nil {
		return []error{notePosition(g.pkg.Fset.Position(pos),
			fmt.Errorf("注入 %s：%v", name, err))}
	}
	params := sig.Params()
	calls, errs := solve(g.pkg.Fset, injectSig.out, params, set)
	if len(errs) > 0 {
		return mapErrors(errs, func(e error) error {
			if w, ok := e.(*diErr); ok {
				return notePosition(w.position, fmt.Errorf("注入 %s：%v", name, w.error))
			}
			return notePosition(g.pkg.Fset.Position(pos), fmt.Errorf("注入 %s：%v", name, e))
		})
	}
	type pendingVar struct {
		name     string
		expr     ast.Expr
		typeInfo *types.Info
	}
	var pendingVars []pendingVar
	ec := new(errorCollector)
	for i := range calls {
		c := &calls[i]
		if c.hasCleanup && !injectSig.cleanup {
			ts := types.TypeString(c.out, nil)
			ec.add(notePosition(
				g.pkg.Fset.Position(pos),
				fmt.Errorf("注入 %s：类型 %s 的 provider 返回清理函数，但注入未返回清理函数", name, ts)))
		}
		if c.hasErr && !injectSig.err {
			ts := types.TypeString(c.out, nil)
			ec.add(notePosition(
				g.pkg.Fset.Position(pos),
				fmt.Errorf("注入 %s：类型 %s 的 provider 返回 error，但注入不允许失败", name, ts)))
		}
		if c.kind == valueExpr {
			if err := accessibleFrom(c.valueTypeInfo, c.valueExpr, g.pkg.PkgPath); err != nil {
				// TODO(light): 显示值表达式的行号。
				ts := types.TypeString(c.out, nil)
				ec.add(notePosition(
					g.pkg.Fset.Position(pos),
					fmt.Errorf("注入 %s：值 %s 无法使用：%v", name, ts, err)))
			}
			if g.values[c.valueExpr] == "" {
				t := c.valueTypeInfo.TypeOf(c.valueExpr)

				name := typeVariableName(t, "", func(name string) string { return "_di" + export(name) + "Value" }, g.nameInFileScope)
				g.values[c.valueExpr] = name
				pendingVars = append(pendingVars, pendingVar{
					name:     name,
					expr:     c.valueExpr,
					typeInfo: c.valueTypeInfo,
				})
			}
		}
	}
	if len(ec.errors) > 0 {
		return ec.errors
	}

	// 先执行一遍收集所有导入，再执行真正的生成。
	injectPass(name, sig, calls, set, doc, &injectorGen{
		g:       g,
		errVar:  disambiguate("err", g.nameInFileScope),
		discard: true,
	})
	injectPass(name, sig, calls, set, doc, &injectorGen{
		g:       g,
		errVar:  disambiguate("err", g.nameInFileScope),
		discard: false,
	})
	if len(pendingVars) > 0 {
		g.p("var (\n")
		for _, pv := range pendingVars {
			g.p("\t%s = ", pv.name)
			g.writeAST(pv.typeInfo, pv.expr)
			g.p("\n")
		}
		g.p(")\n\n")
	}
	return nil
}

// rewritePkgRefs 将 AST 中的包引用重写为对生成包
// 的引用。
func (g *gen) rewritePkgRefs(info *types.Info, node ast.Node) ast.Node {
	start, end := node.Pos(), node.End()
	node = copyAST(node)
	// 首先重写所有包名，从而得知所有
	// 可能冲突的标识符。
	node = astutil.Apply(node, func(c *astutil.Cursor) bool {
		switch node := c.Node().(type) {
		case *ast.Ident:
			// 这是非限定标识符（限定标识符在下面被剥离）。
			obj := info.ObjectOf(node)
			if obj == nil {
				return false
			}
			if pkg := obj.Pkg(); pkg != nil && obj.Parent() == pkg.Scope() && pkg.Path() != g.pkg.PkgPath {
				// 来自点导入或从不同包读取的标识符。
				newPkgID := g.qualifyImport(pkg.Name(), pkg.Path())
				c.Replace(&ast.SelectorExpr{
					X:   ast.NewIdent(newPkgID),
					Sel: ast.NewIdent(node.Name),
				})
				return false
			}
			return true
		case *ast.SelectorExpr:
			pkgIdent, ok := node.X.(*ast.Ident)
			if !ok {
				return true
			}
			pkgName, ok := info.ObjectOf(pkgIdent).(*types.PkgName)
			if !ok {
				return true
			}
			// 这是限定标识符。重写它，并避免访问其子表达式。
			imported := pkgName.Imported()
			newPkgID := g.qualifyImport(imported.Name(), imported.Path())
			c.Replace(&ast.SelectorExpr{
				X:   ast.NewIdent(newPkgID),
				Sel: ast.NewIdent(node.Sel.Name),
			})
			return false
		default:
			return true
		}
	}, nil)
	// 现在我们有了所有标识符，重命名此作用域中声明的变量
	// 以避免冲突。
	newNames := make(map[types.Object]string)
	inNewNames := func(n string) bool {
		for _, other := range newNames {
			if other == n {
				return true
			}
		}
		return false
	}
	var scopeStack []*types.Scope
	pkgScope := g.pkg.Types.Scope()
	node = astutil.Apply(node, func(c *astutil.Cursor) bool {
		if scope := info.Scopes[c.Node()]; scope != nil {
			scopeStack = append(scopeStack, scope)
		}
		id, ok := c.Node().(*ast.Ident)
		if !ok {
			return true
		}
		obj := info.ObjectOf(id)
		if obj == nil {
			// 我们之前已重写此标识符，因此无需
			// 再次重写。
			return true
		}
		if n, ok := newNames[obj]; ok {
			// 我们为这个符号选了新名字。重写它。
			c.Replace(ast.NewIdent(n))
			return false
		}
		if par := obj.Parent(); par == nil || par == pkgScope {
			// 不要重命名方法、字段名或顶层标识符。
			return true
		}

		// 重命名 rewritePkgRefs 的节点内定义的、与生成文件中符号
		// 冲突的符号。
		objName := obj.Name()
		if pos := obj.Pos(); pos < start || end <= pos || !(g.nameInFileScope(objName) || inNewNames(objName)) {
			return true
		}
		newName := disambiguate(objName, func(n string) bool {
			if g.nameInFileScope(n) || inNewNames(n) {
				return true
			}
			if len(scopeStack) > 0 {
				// 避免选择与当前作用域中其他名字
				// 冲突的名字。
				_, obj := scopeStack[len(scopeStack)-1].LookupParent(n, token.NoPos)
				if obj != nil {
					return true
				}
			}
			return false
		})
		newNames[obj] = newName
		c.Replace(ast.NewIdent(newName))
		return false
	}, func(c *astutil.Cursor) bool {
		if info.Scopes[c.Node()] != nil {
			// 应为栈顶；弹出它。
			scopeStack = scopeStack[:len(scopeStack)-1]
		}
		return true
	})
	return node
}

// writeAST 将 AST 节点打印到生成结果中，并重写
// 其遇到的包引用。
func (g *gen) writeAST(info *types.Info, node ast.Node) {
	node = g.rewritePkgRefs(info, node)
	if err := printer.Fprint(&g.buf, g.pkg.Fset, node); err != nil {
		panic(err)
	}
}

func (g *gen) qualifiedID(pkgName, pkgPath, sym string) string {
	name := g.qualifyImport(pkgName, pkgPath)
	if name == "" {
		return sym
	}
	return name + "." + sym
}

func (g *gen) qualifyImport(name, path string) string {
	if path == g.pkg.PkgPath {
		return ""
	}
	// TODO(light): 这里依赖当前加载器的实现细节。
	const vendorPart = "vendor/"
	unvendored := path
	if i := strings.LastIndex(path, vendorPart); i != -1 && (i == 0 || path[i-1] == '/') {
		unvendored = path[i+len(vendorPart):]
	}
	if info, ok := g.imports[unvendored]; ok {
		return info.name
	}
	// TODO(light): 使用导入路径的各部分来消歧。
	newName := disambiguate(name, func(n string) bool {
		// 别让导入占用 "err" 这个名字，那会很烦。
		return n == "err" || g.nameInFileScope(n)
	})
	g.imports[unvendored] = importInfo{
		name:    newName,
		differs: newName != name,
	}
	return newName
}

func (g *gen) nameInFileScope(name string) bool {
	for _, other := range g.imports {
		if other.name == name {
			return true
		}
	}
	for _, other := range g.values {
		if other == name {
			return true
		}
	}
	_, obj := g.pkg.Types.Scope().LookupParent(name, token.NoPos)
	return obj != nil
}

func (g *gen) qualifyPkg(pkg *types.Package) string {
	return g.qualifyImport(pkg.Name(), pkg.Path())
}

func (g *gen) p(format string, args ...interface{}) {
	fmt.Fprintf(&g.buf, format, args...)
}

// injectorGen 是每个注入器的生成状态。
type injectorGen struct {
	g *gen

	paramNames   []string
	localNames   []string
	cleanupNames []string
	errVar       string

	// discard 使 ig.p 和 ig.writeAST 变为空操作。用于为填充
	// g.imports 等副作用而运行生成。
	discard bool
}

// injectPass 根据分析结果生成注入器。
// 传入的 sig 应已被校验。
func injectPass(name string, sig *types.Signature, calls []call, set *ProviderSet, doc *ast.CommentGroup, ig *injectorGen) {
	params := sig.Params()
	injectSig, err := funcOutput(sig)
	if err != nil {
		// 这应该已由调用方检查过。
		panic(err)
	}
	if doc != nil {
		for _, c := range doc.List {
			ig.p("%s\n", c.Text)
		}
	}
	ig.p("func %s(", name)
	for i := 0; i < params.Len(); i++ {
		if i > 0 {
			ig.p(", ")
		}
		pi := params.At(i)
		a := pi.Name()
		if a == "" || a == "_" {
			a = typeVariableName(pi.Type(), "arg", unexport, ig.nameInInjector)
		} else {
			a = disambiguate(a, ig.nameInInjector)
		}
		ig.paramNames = append(ig.paramNames, a)
		if sig.Variadic() && i == params.Len()-1 {
			// 若注入器是可变参数函数，则最后一个参数
			// 保留可变参数签名而非切片。
			ig.p("%s ...%s", ig.paramNames[i], types.TypeString(pi.Type().(*types.Slice).Elem(), ig.g.qualifyPkg))
		} else {
			ig.p("%s %s", ig.paramNames[i], types.TypeString(pi.Type(), ig.g.qualifyPkg))
		}
	}
	outTypeString := types.TypeString(injectSig.out, ig.g.qualifyPkg)
	switch {
	case injectSig.cleanup && injectSig.err:
		ig.p(") (%s, func(), error) {\n", outTypeString)
	case injectSig.cleanup:
		ig.p(") (%s, func()) {\n", outTypeString)
	case injectSig.err:
		ig.p(") (%s, error) {\n", outTypeString)
	default:
		ig.p(") %s {\n", outTypeString)
	}
	for i := range calls {
		c := &calls[i]
		lname := typeVariableName(c.out, "v", unexport, ig.nameInInjector)
		ig.localNames = append(ig.localNames, lname)
		switch c.kind {
		case structProvider:
			ig.structProviderCall(lname, c)
		case funcProviderCall:
			ig.funcProviderCall(lname, c, injectSig)
		case valueExpr:
			ig.valueExpr(lname, c)
		case selectorExpr:
			ig.fieldExpr(lname, c)
		case sliceExpr:
			ig.sliceExpr(lname, c)
		default:
			panic("未知的调用类型")
		}
	}
	if len(calls) == 0 {
		ig.p("\treturn %s", ig.paramNames[set.For(injectSig.out).Arg().Index])
	} else {
		ig.p("\treturn %s", ig.localNames[len(calls)-1])
	}
	if injectSig.cleanup {
		ig.p(", func() {\n")
		for i := len(ig.cleanupNames) - 1; i >= 0; i-- {
			ig.p("\t\t%s()\n", ig.cleanupNames[i])
		}
		ig.p("\t}")
	}
	if injectSig.err {
		ig.p(", nil")
	}
	ig.p("\n}\n\n")
}

func (ig *injectorGen) funcProviderCall(lname string, c *call, injectSig outputSignature) {
	ig.p("\t%s", lname)
	prevCleanup := len(ig.cleanupNames)
	if c.hasCleanup {
		cname := disambiguate("cleanup", ig.nameInInjector)
		ig.cleanupNames = append(ig.cleanupNames, cname)
		ig.p(", %s", cname)
	}
	if c.hasErr {
		ig.p(", %s", ig.errVar)
	}
	ig.p(" := ")
	ig.p("%s", ig.g.qualifiedID(c.pkg.Name(), c.pkg.Path(), c.name))
	if len(c.typeArgs) > 0 {
		ig.p("[")
		for i, ta := range c.typeArgs {
			if i > 0 {
				ig.p(", ")
			}
			ig.p("%s", types.TypeString(ta, ig.g.qualifyPkg))
		}
		ig.p("]")
	}
	ig.p("(")
	for i, a := range c.args {
		if i > 0 {
			ig.p(", ")
		}
		if a < len(ig.paramNames) {
			ig.p("%s", ig.paramNames[a])
		} else {
			ig.p("%s", ig.localNames[a-len(ig.paramNames)])
		}
	}
	if c.varargs {
		ig.p("...")
	}
	ig.p(")\n")
	if c.hasErr {
		ig.p("\tif %s != nil {\n", ig.errVar)
		for i := prevCleanup - 1; i >= 0; i-- {
			ig.p("\t\t%s()\n", ig.cleanupNames[i])
		}
		ig.p("\t\treturn %s", zeroValue(injectSig.out, ig.g.qualifyPkg))
		if injectSig.cleanup {
			ig.p(", nil")
		}
		// TODO(light): 提供失败 provider 的信息。
		ig.p(", err\n")
		ig.p("\t}\n")
	}
}

func (ig *injectorGen) structProviderCall(lname string, c *call) {
	ig.p("\t%s", lname)
	ig.p(" := ")
	base := c.out
	if _, ok := base.(*types.Pointer); ok {
		ig.p("&")
		base = base.(*types.Pointer).Elem()
	}
	ig.p("%s", ig.g.qualifiedID(c.pkg.Name(), c.pkg.Path(), c.name))
	if named, ok := base.(*types.Named); ok && named.TypeArgs().Len() > 0 {
		ig.p("[")
		for i := 0; i < named.TypeArgs().Len(); i++ {
			if i > 0 {
				ig.p(", ")
			}
			ig.p("%s", types.TypeString(named.TypeArgs().At(i), ig.g.qualifyPkg))
		}
		ig.p("]")
	}
	ig.p("{\n")
	for i, a := range c.args {
		ig.p("\t\t%s: ", c.fieldNames[i])
		if a < len(ig.paramNames) {
			ig.p("%s", ig.paramNames[a])
		} else {
			ig.p("%s", ig.localNames[a-len(ig.paramNames)])
		}
		ig.p(",\n")
	}
	ig.p("\t}\n")
}

func (ig *injectorGen) valueExpr(lname string, c *call) {
	ig.p("\t%s := %s\n", lname, ig.g.values[c.valueExpr])
}

func (ig *injectorGen) fieldExpr(lname string, c *call) {
	a := c.args[0]
	ig.p("\t%s := ", lname)
	if c.ptrToField {
		ig.p("&")
	}
	if a < len(ig.paramNames) {
		ig.p("%s.%s\n", ig.paramNames[a], c.name)
	} else {
		ig.p("%s.%s\n", ig.localNames[a-len(ig.paramNames)], c.name)
	}
}

func (ig *injectorGen) sliceExpr(lname string, c *call) {
	ig.p("\t%s := %s{", lname, types.TypeString(c.out, ig.g.qualifyPkg))
	for i, a := range c.args {
		if i > 0 {
			ig.p(", ")
		}
		if a < len(ig.paramNames) {
			ig.p("%s", ig.paramNames[a])
		} else {
			ig.p("%s", ig.localNames[a-len(ig.paramNames)])
		}
	}
	ig.p("}\n")
}

// nameInInjector 判断 name 是否与当前注入器中
// 的其他标识符冲突。
func (ig *injectorGen) nameInInjector(name string) bool {
	if name == ig.errVar {
		return true
	}
	for _, a := range ig.paramNames {
		if a == name {
			return true
		}
	}
	for _, l := range ig.localNames {
		if l == name {
			return true
		}
	}
	for _, l := range ig.cleanupNames {
		if l == name {
			return true
		}
	}
	return ig.g.nameInFileScope(name)
}

func (ig *injectorGen) p(format string, args ...interface{}) {
	if ig.discard {
		return
	}
	ig.g.p(format, args...)
}

// zeroValue 返回求值为给定类型零值的
// 最短表达式。
func zeroValue(t types.Type, qf types.Qualifier) string {
	switch u := t.Underlying().(type) {
	case *types.Array, *types.Struct:
		return types.TypeString(t, qf) + "{}"
	case *types.Basic:
		info := u.Info()
		switch {
		case info&types.IsBoolean != 0:
			return "false"
		case info&(types.IsInteger|types.IsFloat|types.IsComplex) != 0:
			return "0"
		case info&types.IsString != 0:
			return `""`
		default:
			panic("不可达")
		}
	case *types.Chan, *types.Interface, *types.Map, *types.Pointer, *types.Signature, *types.Slice:
		return "nil"
	default:
		panic("不可达")
	}
}

// typeVariableName 根据类型名生成一个去重后的变量名。
// 若无法从类型派生名字，则使用 defaultName。
// transform 用于变换派生出的名字（包括 defaultName）；
// 常用函数包括 export 和 unexport。
// collides 用于判断名字是否冲突。若派生的名字中任意一个
// 无歧义，则使用它；否则用 disambiguate() 对
// 第一个派生的名字去重。
func typeVariableName(t types.Type, defaultName string, transform func(string) string, collides func(string) bool) string {
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	var names []string
	switch t := t.(type) {
	case *types.Basic:
		if t.Name() != "" {
			names = append(names, t.Name())
		}
	case *types.Named:
		obj := t.Obj()
		if name := obj.Name(); name != "" {
			names = append(names, name)
		}
		// 若可能，提供一个带包名前缀的备选名字。
		// 例如，冲突时用 "fooCfg" 而不是 "cfg2"。
		if pkg := obj.Pkg(); pkg != nil && pkg.Name() != "" {
			names = append(names, fmt.Sprintf("%s%s", pkg.Name(), strings.Title(obj.Name())))
		}
	}

	// 若无法派生出名字，使用 defaultName。
	if len(names) == 0 {
		names = append(names, defaultName)
	}

	// 变换名字。
	for i, name := range names {
		names[i] = transform(name)
	}

	// 检查是否存在无歧义的名字；若有则使用它。
	for _, name := range names {
		if !token.Lookup(name).IsKeyword() && !collides(name) {
			return name
		}
	}
	// 否则对第一个名字去重。
	return disambiguate(names[0], collides)
}

// unexport 将可能导出的名字转换为未导出的名字。
func unexport(name string) string {
	if name == "" {
		return ""
	}
	r, sz := utf8.DecodeRuneInString(name)
	if !unicode.IsUpper(r) {
		// foo -> foo
		return name
	}
	r2, sz2 := utf8.DecodeRuneInString(name[sz:])
	if !unicode.IsUpper(r2) {
		// Foo -> foo
		return string(unicode.ToLower(r)) + name[sz:]
	}
	// UPPERWord -> upperWord
	sbuf := new(strings.Builder)
	sbuf.WriteRune(unicode.ToLower(r))
	i := sz
	r, sz = r2, sz2
	for unicode.IsUpper(r) && sz > 0 {
		r2, sz2 := utf8.DecodeRuneInString(name[i+sz:])
		if sz2 > 0 && unicode.IsLower(r2) {
			break
		}
		i += sz
		sbuf.WriteRune(unicode.ToLower(r))
		r, sz = r2, sz2
	}
	sbuf.WriteString(name[i:])
	return sbuf.String()
}

// export 将可能未导出的名字转换为导出的名字。
func export(name string) string {
	if name == "" {
		return ""
	}
	r, sz := utf8.DecodeRuneInString(name)
	if unicode.IsUpper(r) {
		// Foo -> Foo
		return name
	}
	// fooBar -> FooBar
	sbuf := new(strings.Builder)
	sbuf.WriteRune(unicode.ToUpper(r))
	sbuf.WriteString(name[sz:])
	return sbuf.String()
}

// disambiguate 选取一个唯一的名字；若 name 已唯一则优先使用它。
// 它还会针对 Go 的保留关键字进行消歧。
func disambiguate(name string, collides func(string) bool) string {
	if !token.Lookup(name).IsKeyword() && !collides(name) {
		return name
	}
	buf := []byte(name)
	if len(buf) > 0 && buf[len(buf)-1] >= '0' && buf[len(buf)-1] <= '9' {
		buf = append(buf, '_')
	}
	base := len(buf)
	for n := 2; ; n++ {
		buf = strconv.AppendInt(buf[:base], int64(n), 10)
		sbuf := string(buf)
		if !token.Lookup(sbuf).IsKeyword() && !collides(sbuf) {
			return sbuf
		}
	}
}

// accessibleFrom 判断 node 能否在不违反 Go 可见性规则
// 的情况下被复制到 wantPkg。
func accessibleFrom(info *types.Info, node ast.Node, wantPkg string) error {
	var unexportError error
	ast.Inspect(node, func(node ast.Node) bool {
		if unexportError != nil {
			return false
		}
		ident, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		obj := info.ObjectOf(ident)
		if _, ok := obj.(*types.PkgName); ok {
			// 本地包名没问题，因为我们可以直接重新导入它们。
			return true
		}
		if pkg := obj.Pkg(); pkg != nil {
			if !ast.IsExported(ident.Name) && pkg.Path() != wantPkg {
				unexportError = fmt.Errorf("使用了未导出的标识符 %s", obj.Name())
				return false
			}
			if obj.Parent() != nil && obj.Parent() != pkg.Scope() {
				unexportError = fmt.Errorf("%s 未在包作用域中声明", obj.Name())
				return false
			}
		}
		return true
	})
	return unexportError
}

var (
	errorType   = types.Universe.Lookup("error").Type()
	cleanupType = types.NewSignature(nil, nil, nil, false)
)
