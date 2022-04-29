package main

// This file handles the app code introspection.
// It catalogs the controllers, their methods, and their arguments.

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/roblillack/mars"
)

// SourceInfo is the top-level struct containing all extracted information
// about the app source code, used to generate main.go.
type SourceInfo struct {
	PackageName string
	// StructSpecs lists type info for all structs found under the code paths.
	// They may be queried to determine which ones (transitively) embed certain types.
	StructSpecs []*TypeInfo
	// A list of import paths.
	// Revel notices files with an init() function and imports that package.
	InitImportPaths []string

	// controllerSpecs lists type info for all structs found under
	// app/controllers/... that embed (directly or indirectly) mars.Controller
	controllerSpecs []*TypeInfo
}

// TypeInfo summarizes information about a struct type in the app source code.
type TypeInfo struct {
	StructName  string // e.g. "Application"
	ImportPath  string // e.g. "github.com/mars/samples/chat/app/controllers"
	PackageName string // e.g. "controllers"
	MethodSpecs []*MethodSpec

	// Used internally to identify controllers that indirectly embed *mars.Controller.
	embeddedTypes []*embeddedTypeName
}

func (info TypeInfo) String() string {
	str := fmt.Sprintf("%s => %s.%s", info.ImportPath, info.PackageName, info.StructName)
	for _, i := range info.MethodSpecs {
		str += "\n  - " + i.String()
	}
	return str
}

type TypeInfosByName []*TypeInfo

func (slice TypeInfosByName) Len() int {
	return len(slice)
}

func (slice TypeInfosByName) Less(i, j int) bool {
	return slice[i].StructName < slice[j].StructName
}

func (slice TypeInfosByName) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// MethodSpec represents a method defined for a receiver type represented by TypeInfo.
type MethodSpec struct {
	Name string       // Name of the method, e.g. "Index"
	Args []*MethodArg // Argument descriptors
}

func (m MethodSpec) String() string {
	str := fmt.Sprintf("%s(", m.Name)
	for idx, i := range m.Args {
		if idx > 0 {
			str += ", "
		}
		str += fmt.Sprintf("%s %s", i.Name, i.TypeExpr.Expr)
	}
	return str + ")"
}

// MethodArg represents a single argument to a method represented by MethodSpec.
type MethodArg struct {
	Name       string   // Name of the argument.
	TypeExpr   TypeExpr // The name of the type, e.g. "int", "*pkg.UserType"
	ImportPath string   // If the arg is of an imported type, this is the import path.
}

type embeddedTypeName struct {
	ImportPath, StructName string
}

// Maps a controller simple name (e.g. "Login") to the methods for which it is a
// receiver.
type methodMap map[string][]*MethodSpec

// ProcessSource parses the app's controllers directory and return a list of
// the controller types found. Returns a CompileError if the parsing fails.
func ProcessSource(path string, verbose bool) (*SourceInfo, error) {
	// Parse files within the path.
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, path, func(f os.FileInfo) bool {
		return !f.IsDir() && !strings.HasPrefix(f.Name(), ".") && strings.HasSuffix(f.Name(), ".go")
	}, 0)
	if err != nil {
		return nil, err
	}

	// If there is no code in this directory, skip it.
	if len(pkgs) == 0 {
		return nil, nil
	} else if len(pkgs) > 1 {
		return nil, fmt.Errorf("Most unexpected! Multiple packages in a single directory: %s", path)
	}

	var pkg *ast.Package
	for _, v := range pkgs {
		pkg = v
	}

	return processPackage(fset, pkg.Name, path, pkg, verbose), nil
}

// ProcessFile created a SourceInfo data structure similarly to ProcessSource,
// but for a single file.
func ProcessFile(fset *token.FileSet, fileName string, file *ast.File) *SourceInfo {
	pkg := &ast.Package{
		Name:  file.Name.Name,
		Files: map[string]*ast.File{fileName: file},
	}

	return processPackage(fset, file.Name.Name, filepath.Dir(fileName), pkg, false)
}

func processPackage(fset *token.FileSet, pkgImportPath, pkgPath string, pkg *ast.Package, verbose bool) *SourceInfo {
	var (
		structSpecs     []*TypeInfo
		initImportPaths []string

		methodSpecs = make(methodMap)
	)

	// For each source file in the package...
	for _, fInfo := range getSortedFiles(pkg) {
		if verbose {
			fmt.Println(fInfo.Filename)
		}

		// Imports maps the package key to the full import path.
		// e.g. import "sample/app/models" => "models": "sample/app/models"
		imports := map[string]string{}

		// For each declaration in the source file...
		for _, decl := range fInfo.File.Decls {
			addImports(imports, decl, pkgPath)

			// Match and add both structs and methods
			structSpecs = appendStruct(structSpecs, pkgImportPath, pkg, decl, imports, fset)
			appendAction(fset, methodSpecs, decl, pkgImportPath, pkg.Name, imports)
		}
	}

	// Add the method specs to the struct specs.
	for _, spec := range structSpecs {
		spec.MethodSpecs = methodSpecs[spec.StructName]
	}

	return &SourceInfo{
		PackageName:     pkg.Name,
		StructSpecs:     structSpecs,
		InitImportPaths: initImportPaths,
	}
}

// getFuncName returns a name for this func or method declaration.
// e.g. "(*Application).SayHello" for a method, "SayHello" for a func.
func getFuncName(funcDecl *ast.FuncDecl) string {
	prefix := ""
	if funcDecl.Recv != nil {
		recvType := funcDecl.Recv.List[0].Type
		if recvStarType, ok := recvType.(*ast.StarExpr); ok {
			prefix = "(*" + recvStarType.X.(*ast.Ident).Name + ")"
		} else {
			prefix = recvType.(*ast.Ident).Name
		}
		prefix += "."
	}
	return prefix + funcDecl.Name.Name
}

// full path --> package name
var importsCache = map[string]string{}

func addImports(imports map[string]string, decl ast.Decl, srcDir string) {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok {
		return
	}

	if genDecl.Tok != token.IMPORT {
		return
	}

	for _, spec := range genDecl.Specs {
		importSpec := spec.(*ast.ImportSpec)
		var pkgAlias string
		if importSpec.Name != nil {
			pkgAlias = importSpec.Name.Name
			if pkgAlias == "_" {
				continue
			}
		}
		quotedPath := importSpec.Path.Value           // e.g. "\"sample/app/models\""
		fullPath := quotedPath[1 : len(quotedPath)-1] // Remove the quotes

		if pkgAlias != "" {
			imports[pkgAlias] = fullPath
			continue
		}

		if n, ok := importsCache[fullPath]; ok {
			imports[n] = fullPath
			continue
		}

		if !strings.Contains(fullPath, ".") {
			// go standard library packages ==> package will always match directory name
			imports[fullPath[strings.Index(fullPath, "/")+1:]] = fullPath
			continue
		}

		if fullPath == mars.MarsImportPath {
			// Don't expect Mars to be resolvable during code generation …
			imports["mars"] = mars.MarsImportPath
			continue
		}

		// If the package was not aliased (common case) and is not part of the standard library,
		// we need to import it to figure out what the package name is.
		// TODO: We can improve performance here a bit:
		//       Use build.FindOnly and then use parser.ParseDir with mode PackageClauseOnly
		pkg, err := build.Import(fullPath, srcDir, 0)
		if err != nil {
			// We expect this to happen for apps using reverse routing (since we
			// have not yet generated the routes).  Don't log that.
			if !strings.HasSuffix(fullPath, "/app/routes") {
				mars.TRACE.Println("Could not find import:", fullPath)
			}
			continue
		}
		pkgAlias = pkg.Name
		imports[pkgAlias] = fullPath

		// ok, spare us the search next time
		importsCache[fullPath] = pkgAlias
	}
}

// If this Decl is a struct type definition, it is summarized and added to specs.
// Else, specs is returned unchanged.
func appendStruct(specs []*TypeInfo, pkgImportPath string, pkg *ast.Package, decl ast.Decl, imports map[string]string, fset *token.FileSet) []*TypeInfo {
	// Filter out non-Struct type declarations.
	spec, found := getStructTypeDecl(decl, fset)
	if !found {
		return specs
	}
	structType := spec.Type.(*ast.StructType)

	// At this point we know it's a type declaration for a struct.
	// Fill in the rest of the info by diving into the fields.
	// Add it provisionally to the Controller list -- it's later filtered using field info.
	controllerSpec := &TypeInfo{
		StructName:  spec.Name.Name,
		ImportPath:  pkgImportPath,
		PackageName: pkg.Name,
	}

	for _, field := range structType.Fields.List {
		// If field.Names is set, it's not an embedded type.
		if field.Names != nil {
			continue
		}

		// A direct "sub-type" has an ast.Field as either:
		//   Ident { "AppController" }
		//   SelectorExpr { "rev", "Controller" }
		// Additionally, that can be wrapped by StarExprs.
		fieldType := field.Type
		pkgName, typeName := func() (string, string) {
			// Drill through any StarExprs.
			for {
				if starExpr, ok := fieldType.(*ast.StarExpr); ok {
					fieldType = starExpr.X
					continue
				}
				break
			}

			// If the embedded type is in the same package, it's an Ident.
			if ident, ok := fieldType.(*ast.Ident); ok {
				return "", ident.Name
			}

			if selectorExpr, ok := fieldType.(*ast.SelectorExpr); ok {
				if pkgIdent, ok := selectorExpr.X.(*ast.Ident); ok {
					return pkgIdent.Name, selectorExpr.Sel.Name
				}
			}
			return "", ""
		}()

		// If a typename wasn't found, skip it.
		if typeName == "" {
			continue
		}

		// Find the import path for this type.
		// If it was referenced without a package name, use the current package import path.
		// Else, look up the package's import path by name.
		var importPath string
		if pkgName == "" {
			importPath = pkgImportPath
		} else {
			var ok bool
			if importPath, ok = imports[pkgName]; !ok {
				log.Print("Failed to find import path for ", pkgName, ".", typeName)
				continue
			}
		}

		controllerSpec.embeddedTypes = append(controllerSpec.embeddedTypes, &embeddedTypeName{
			ImportPath: importPath,
			StructName: typeName,
		})
	}

	return append(specs, controllerSpec)
}

// If decl is a Method declaration, it is summarized and added to the array
// underneath its receiver type.
// e.g. "Login" => {MethodSpec, MethodSpec, ..}
func appendAction(fset *token.FileSet, mm methodMap, decl ast.Decl, pkgImportPath, pkgName string, imports map[string]string) {
	// Func declaration?
	funcDecl, ok := decl.(*ast.FuncDecl)
	if !ok {
		return
	}

	// Have a receiver?
	if funcDecl.Recv == nil {
		return
	}

	// Is it public?
	if !funcDecl.Name.IsExported() {
		return
	}

	// Does it return a single Result?
	if funcDecl.Type.Results == nil || len(funcDecl.Type.Results.List) != 1 {
		return
	}
	selExpr, ok := funcDecl.Type.Results.List[0].Type.(*ast.SelectorExpr)
	if !ok {
		return
	}
	if selExpr.Sel.Name != "Result" {
		return
	}
	if pkgIdent, ok := selExpr.X.(*ast.Ident); !ok || imports[pkgIdent.Name] != mars.MarsImportPath {
		return
	}

	method := &MethodSpec{
		Name: funcDecl.Name.Name,
	}

	// Add a description of the arguments to the method.
	for _, field := range funcDecl.Type.Params.List {
		for _, name := range field.Names {
			var importPath string
			typeExpr := NewTypeExpr(pkgName, field.Type)
			if !typeExpr.Valid {
				log.Printf("Didn't understand argument '%s' of action %s. Ignoring.\n", name, getFuncName(funcDecl))
				return // We didn't understand one of the args.  Ignore this action.
			}
			if typeExpr.PkgName != "" {
				var ok bool
				if importPath, ok = imports[typeExpr.PkgName]; !ok {
					log.Println("Failed to find import for arg of type:", typeExpr.TypeName(""))
				}
			}
			method.Args = append(method.Args, &MethodArg{
				Name:       name.Name,
				TypeExpr:   typeExpr,
				ImportPath: importPath,
			})
		}
	}

	var recvTypeName string
	recvType := funcDecl.Recv.List[0].Type
	if recvStarType, ok := recvType.(*ast.StarExpr); ok {
		recvTypeName = recvStarType.X.(*ast.Ident).Name
	} else {
		recvTypeName = recvType.(*ast.Ident).Name
	}

	mm[recvTypeName] = append(mm[recvTypeName], method)
}

func (s *embeddedTypeName) String() string {
	return s.ImportPath + "." + s.StructName
}

// getStructTypeDecl checks if the given decl is a type declaration for a
// struct.  If so, the TypeSpec is returned.
func getStructTypeDecl(decl ast.Decl, fset *token.FileSet) (spec *ast.TypeSpec, found bool) {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok {
		return
	}

	if genDecl.Tok != token.TYPE {
		return
	}

	if len(genDecl.Specs) == 0 {
		mars.WARN.Printf("Surprising: %s:%d Decl contains no specifications", fset.Position(decl.Pos()).Filename, fset.Position(decl.Pos()).Line)
		return
	}

	spec = genDecl.Specs[0].(*ast.TypeSpec)
	_, found = spec.Type.(*ast.StructType)

	return
}

func containsString(list []string, target string) bool {
	for _, el := range list {
		if el == target {
			return true
		}
	}
	return false
}

// TypesThatEmbed returns all types that (directly or indirectly) embed the
// target type, which must be a fully qualified type name,
// e.g. "github.com/roblillack/mars.Controller"
func (s *SourceInfo) TypesThatEmbed(targetType string) (filtered []*TypeInfo) {
	// Do a search in the "embedded type graph", starting with the target type.
	var (
		nodeQueue = []string{targetType}
		processed []string
	)
	for len(nodeQueue) > 0 {
		controllerSimpleName := nodeQueue[0]
		nodeQueue = nodeQueue[1:]
		processed = append(processed, controllerSimpleName)

		// Look through all known structs.
		for _, spec := range s.StructSpecs {
			// If this one has been processed or is already in nodeQueue, then skip it.
			if containsString(processed, spec.String()) || containsString(nodeQueue, spec.String()) {
				continue
			}

			// Look through the embedded types to see if the current type is among them.
			for _, embeddedType := range spec.embeddedTypes {

				// If so, add this type's simple name to the nodeQueue, and its spec to
				// the filtered list.
				if controllerSimpleName == embeddedType.String() {
					nodeQueue = append(nodeQueue, fmt.Sprintf("%s.%s", spec.PackageName, spec.StructName))
					filtered = append(filtered, spec)
					break
				}
			}
		}
	}
	return
}

// ControllerSpecs returns all types are therefore regarded as controller
// because they (transitively) embed mars.Controller.
func (s *SourceInfo) ControllerSpecs() []*TypeInfo {
	if s.controllerSpecs == nil {
		s.controllerSpecs = s.TypesThatEmbed(mars.MarsImportPath + ".Controller")
	}

	sort.Sort(TypeInfosByName(s.controllerSpecs))
	return s.controllerSpecs
}

// TypeExpr provides a type name that may be rewritten to use a package name.
type TypeExpr struct {
	Expr     string // The unqualified type expression, e.g. "[]*MyType"
	PkgName  string // The default package idenifier
	pkgIndex int    // The index where the package identifier should be inserted.
	Valid    bool
}

func firstNonEmpty(strs ...string) string {
	for _, str := range strs {
		if len(str) > 0 {
			return str
		}
	}
	return ""
}

// TypeName returns the fully-qualified type name for this expression.
// The caller may optionally specify a package name to override the default.
func (e TypeExpr) TypeName(pkgOverride string) string {
	pkgName := firstNonEmpty(pkgOverride, e.PkgName)
	if pkgName == "" {
		return e.Expr
	}
	return e.Expr[:e.pkgIndex] + pkgName + "." + e.Expr[e.pkgIndex:]
}

// CalcImportAliases looks through all the method args and returns a set of
// unique import paths that cover all the method arg types. Additionally,
// assign package aliases when necessary to resolve ambiguity.
func (s *SourceInfo) CalcImportAliases() map[string]string {
	aliases := make(map[string]string)
	for _, spec := range s.ControllerSpecs() {
		//addAlias(aliases, spec.ImportPath, spec.PackageName)

		for _, methSpec := range spec.MethodSpecs {
			for _, methArg := range methSpec.Args {
				if methArg.ImportPath == "" {
					continue
				}
				addAlias(aliases, methArg.ImportPath, methArg.TypeExpr.PkgName)
			}
		}
	}

	// Add the "InitImportPaths", with alias "_"
	for _, importPath := range s.InitImportPaths {
		if _, ok := aliases[importPath]; !ok {
			aliases[importPath] = "_"
		}
	}

	return aliases
}

func addAlias(aliases map[string]string, importPath, pkgName string) {
	if _, ok := aliases[importPath]; ok {
		return
	}

	aliases[importPath] = makePackageAlias(aliases, pkgName)
}

func makePackageAlias(aliases map[string]string, pkgName string) string {
	i := 0
	alias := pkgName
	for containsValue(aliases, alias) {
		alias = fmt.Sprintf("%s%d", pkgName, i)
		i++
	}
	return alias
}

func containsValue(m map[string]string, val string) bool {
	for _, v := range m {
		if v == val {
			return true
		}
	}
	return false
}

// NewTypeExpr returns the syntactic expression for referencing this type in Go.
func NewTypeExpr(pkgName string, expr ast.Expr) TypeExpr {
	switch t := expr.(type) {
	case *ast.Ident:
		if _, ok := builtinTypes[t.Name]; ok {
			pkgName = ""
		}
		return TypeExpr{t.Name, pkgName, 0, true}
	case *ast.InterfaceType:
		return TypeExpr{"interface{}", "", 0, true}
	case *ast.SelectorExpr:
		e := NewTypeExpr(pkgName, t.X)
		return TypeExpr{t.Sel.Name, e.Expr, 0, e.Valid}
	case *ast.StarExpr:
		e := NewTypeExpr(pkgName, t.X)
		return TypeExpr{"*" + e.Expr, e.PkgName, e.pkgIndex + 1, e.Valid}
	case *ast.ArrayType:
		e := NewTypeExpr(pkgName, t.Elt)
		return TypeExpr{"[]" + e.Expr, e.PkgName, e.pkgIndex + 2, e.Valid}
	case *ast.Ellipsis:
		e := NewTypeExpr(pkgName, t.Elt)
		return TypeExpr{"[]" + e.Expr, e.PkgName, e.pkgIndex + 2, e.Valid}
	default:
		log.Printf("Failed to generate name for field: %s. Make sure the field name is valid.\n", reflect.TypeOf(expr))
	}
	return TypeExpr{Valid: false}
}

var builtinTypes = map[string]struct{}{
	"bool":       {},
	"byte":       {},
	"complex128": {},
	"complex64":  {},
	"error":      {},
	"float32":    {},
	"float64":    {},
	"int":        {},
	"int16":      {},
	"int32":      {},
	"int64":      {},
	"int8":       {},
	"rune":       {},
	"string":     {},
	"uint":       {},
	"uint16":     {},
	"uint32":     {},
	"uint64":     {},
	"uint8":      {},
	"uintptr":    {},
	"any":        {},
}
