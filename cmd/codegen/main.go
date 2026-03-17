package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// ServiceInfo holds metadata extracted from a generated _http.pb.go file.
type ServiceInfo struct {
	ModuleName      string // e.g. "project"
	ServiceName     string // e.g. "UserService"
	FeaturePkg      string // e.g. "user"
	FeatureDir      string // e.g. "internal/feature/user"
	GenImportPath   string // e.g. "project/gen/user/v1"
	GenPackageAlias string // e.g. "userv1"
	Methods         []MethodInfo
	HasFacade       bool // true if facade.go exists in the feature dir
}

// MethodInfo holds metadata for a single RPC method.
type MethodInfo struct {
	Name         string // e.g. "CreateUser"
	RequestType  string // e.g. "CreateUserRequest"
	ResponseType string // e.g. "CreateUserResponse"
}

func main() {
	moduleName := readModuleName()
	services := scanGeneratedFiles(moduleName)

	for i := range services {
		// Detect if facade.go exists (for wire.go generation)
		if _, err := os.Stat(services[i].FeatureDir + "/facade.go"); err == nil {
			services[i].HasFacade = true
		}
		generateService(services[i])
		generateUseCase(services[i])
		generateRepo(services[i])
		generateWire(services[i])
	}

	if len(services) == 0 {
		fmt.Println("codegen: no *HTTPServer interfaces found in gen/")
	}
}

// readModuleName reads the module name from go.mod.
func readModuleName() string {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		panic("codegen: cannot read go.mod: " + err.Error())
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	panic("codegen: module name not found in go.mod")
}

// scanGeneratedFiles walks gen/ and parses _http.pb.go files.
func scanGeneratedFiles(moduleName string) []ServiceInfo {
	var services []ServiceInfo

	filepath.Walk("gen", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), "_http.pb.go") {
			return nil
		}

		svc := parseHTTPFile(path, moduleName)
		if svc != nil {
			services = append(services, *svc)
		}
		return nil
	})

	return services
}

// parseHTTPFile parses a _http.pb.go file and extracts the HTTPServer interface.
func parseHTTPFile(path string, moduleName string) *ServiceInfo {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "codegen: parse %s: %v\n", path, err)
		return nil
	}

	genPkgAlias := f.Name.Name // e.g. "userv1"

	// Derive import path from file path: gen/user/v1/ -> project/gen/user/v1
	dir := filepath.ToSlash(filepath.Dir(path))
	genImportPath := moduleName + "/" + dir

	// Derive feature name: gen/user/v1 -> "user"
	parts := strings.Split(dir, "/")
	if len(parts) < 2 {
		return nil
	}
	featurePkg := parts[1] // "user" from "gen/user/v1"

	// Find the *HTTPServer interface
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec := spec.(*ast.TypeSpec)
			name := typeSpec.Name.Name
			if !strings.HasSuffix(name, "HTTPServer") {
				continue
			}
			iface, ok := typeSpec.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}

			serviceName := strings.TrimSuffix(name, "HTTPServer")
			methods := extractMethods(iface)

			return &ServiceInfo{
				ModuleName:      moduleName,
				ServiceName:     serviceName,
				FeaturePkg:      featurePkg,
				FeatureDir:      "internal/feature/" + featurePkg,
				GenImportPath:   genImportPath,
				GenPackageAlias: genPkgAlias,
				Methods:         methods,
			}
		}
	}

	return nil
}

// extractMethods extracts method info from an interface AST node.
func extractMethods(iface *ast.InterfaceType) []MethodInfo {
	var methods []MethodInfo
	for _, method := range iface.Methods.List {
		if len(method.Names) == 0 {
			continue
		}
		funcType, ok := method.Type.(*ast.FuncType)
		if !ok {
			continue
		}

		name := method.Names[0].Name
		reqType := extractTypeName(funcType.Params.List[1].Type)   // 2nd param (after context)
		respType := extractTypeName(funcType.Results.List[0].Type) // 1st result (before error)

		methods = append(methods, MethodInfo{
			Name:         name,
			RequestType:  reqType,
			ResponseType: respType,
		})
	}
	return methods
}

// extractTypeName gets the type name from an AST expression (handles *T).
func extractTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return extractTypeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	default:
		return "unknown"
	}
}

// --- File generation ---

func generateService(svc ServiceInfo) {
	writeFile(svc.FeatureDir+"/service.go", tplService, svc, true)
}

func generateUseCase(svc ServiceInfo) {
	writeFile(svc.FeatureDir+"/usecase.go", tplUseCase, svc, false) // never overwrite
}

func generateRepo(svc ServiceInfo) {
	writeFile(svc.FeatureDir+"/repo.go", tplRepo, svc, false) // never overwrite
}

func generateWire(svc ServiceInfo) {
	writeFile(svc.FeatureDir+"/wire.go", tplWire, svc, true)
}

func writeFile(path string, tpl string, data ServiceInfo, overwrite bool) {
	if !overwrite {
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("codegen: skip %s (already exists)\n", path)
			return
		}
	}

	t := template.Must(template.New("").Parse(tpl))
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		panic(fmt.Sprintf("codegen: template %s: %v", path, err))
	}

	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		panic(fmt.Sprintf("codegen: write %s: %v", path, err))
	}

	action := "wrote"
	if !overwrite {
		action = "created"
	}
	fmt.Printf("codegen: %s %s\n", action, path)
}

// --- Templates ---

var tplService = `// Code generated by codegen. DO NOT EDIT.

package {{.FeaturePkg}}

import (
	"context"

	{{.GenPackageAlias}} "{{.GenImportPath}}"
)

type Service struct {
	{{.GenPackageAlias}}.Unimplemented{{.ServiceName}}Server
	uc *UseCase
}

func NewService(uc *UseCase) *Service {
	return &Service{uc: uc}
}
{{range .Methods}}
func (s *Service) {{.Name}}(ctx context.Context, req *{{$.GenPackageAlias}}.{{.RequestType}}) (*{{$.GenPackageAlias}}.{{.ResponseType}}, error) {
	return s.uc.{{.Name}}(ctx, req)
}
{{end}}`

var tplUseCase = `package {{.FeaturePkg}}

import (
	"context"
	"log/slog"

	{{.GenPackageAlias}} "{{.GenImportPath}}"
)

type UseCase struct {
	repo   *Repo
	logger *slog.Logger
}

func NewUseCase(repo *Repo, logger *slog.Logger) *UseCase {
	return &UseCase{repo: repo, logger: logger}
}
{{range .Methods}}
func (uc *UseCase) {{.Name}}(ctx context.Context, req *{{$.GenPackageAlias}}.{{.RequestType}}) (*{{$.GenPackageAlias}}.{{.ResponseType}}, error) {
	// TODO: implement business logic
	panic("not implemented")
}
{{end}}`

var tplRepo = `package {{.FeaturePkg}}

import (
	"{{.ModuleName}}/gen/query"

	"gorm.io/gorm"
)

type Repo struct {
	db *gorm.DB
	q  *query.Query
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db, q: query.Use(db)}
}
`

var tplWire = `// Code generated by codegen. DO NOT EDIT.

package {{.FeaturePkg}}

import "github.com/google/wire"

var ProviderSet = wire.NewSet(NewRepo, NewUseCase, NewService{{if .HasFacade}}, NewFacade, WireBind{{end}})
`
