package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

var featureNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "scaffold:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("missing subcommand, supported: add-feature")
	}

	switch args[0] {
	case "add-feature":
		return runAddFeature(args[1:])
	default:
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func runAddFeature(args []string) error {
	fs := flag.NewFlagSet("add-feature", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	name := fs.String("name", "", "feature name, for example order")
	root := fs.String("root", ".", "project root")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(*name) == "" {
		return errors.New("feature name is required, use -name order")
	}

	return addFeature(*root, *name)
}

func addFeature(root string, name string) error {
	name = strings.TrimSpace(name)
	if !featureNamePattern.MatchString(name) {
		return fmt.Errorf("invalid feature name %q, only lowercase letters, numbers and underscore are allowed", name)
	}

	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	moduleName, err := readModuleName(filepath.Join(root, "go.mod"))
	if err != nil {
		return err
	}

	data := featureTemplateData{
		FeatureName:        name,
		FeatureNamePascal:  toPascalCase(name),
		FeatureNameUpper:   toUpperSnake(name),
		FeatureRoutePlural: pluralize(name),
		TableName:          pluralize(name),
		ModuleName:         moduleName,
		PackageAlias:       name + "v1",
	}

	if err := createFeatureFiles(root, data); err != nil {
		return err
	}
	if err := ensureBufGenIncludesAPI(root); err != nil {
		return err
	}

	fmt.Printf("scaffold: created feature %s\n", name)
	fmt.Println("next steps:")
	fmt.Println("  1. make generate")
	fmt.Println("  2. make test")
	fmt.Printf("  3. edit api/%s/v1/%s.proto and internal/feature/%s/usecase.go\n", name, name, name)
	return nil
}

type featureTemplateData struct {
	FeatureName        string
	FeatureNamePascal  string
	FeatureNameUpper   string
	FeatureRoutePlural string
	TableName          string
	ModuleName         string
	PackageAlias       string
}

func createFeatureFiles(root string, data featureTemplateData) error {
	apiDir := filepath.Join(root, "api", data.FeatureName, "v1")
	featureDir := filepath.Join(root, "internal", "feature", data.FeatureName)
	portsDir := filepath.Join(root, "internal", "domain", "ports")
	schemaDir := filepath.Join(root, "db", "schema")
	migrationsDir := filepath.Join(root, "db", "migrations")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(featureDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(portsDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		return err
	}

	migrationName, err := nextMigrationName(migrationsDir, data.TableName)
	if err != nil {
		return err
	}

	files := []struct {
		path string
		body string
	}{
		{
			path: filepath.Join(apiDir, data.FeatureName+".proto"),
			body: renderTemplate(protoTemplate, data),
		},
		{
			path: filepath.Join(apiDir, "error_reason.proto"),
			body: renderTemplate(errorReasonTemplate, data),
		},
		{
			path: filepath.Join(portsDir, data.FeatureName+".go"),
			body: renderTemplate(portsTemplate, data),
		},
		{
			path: filepath.Join(featureDir, "facade.go"),
			body: renderTemplate(facadeTemplate, data),
		},
		{
			path: filepath.Join(featureDir, "wire_bind.go"),
			body: renderTemplate(wireBindTemplate, data),
		},
		{
			path: filepath.Join(schemaDir, data.TableName+".sql"),
			body: renderTemplate(schemaTemplate, data),
		},
		{
			path: filepath.Join(migrationsDir, migrationName),
			body: renderTemplate(migrationTemplate, data),
		},
	}

	for _, file := range files {
		if _, err := os.Stat(file.path); err == nil {
			return fmt.Errorf("file already exists: %s", file.path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	for _, file := range files {
		if err := os.WriteFile(file.path, []byte(file.body), 0o644); err != nil {
			return err
		}
	}

	return nil
}

func renderTemplate(tpl string, data featureTemplateData) string {
	t := template.Must(template.New("feature").Parse(tpl))
	var builder strings.Builder
	if err := t.Execute(&builder, data); err != nil {
		panic(err)
	}
	return builder.String()
}

func ensureBufGenIncludesAPI(root string) error {
	path := filepath.Join(root, "buf.gen.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var cfg bufGenFile
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}

	if len(cfg.Inputs) == 0 {
		cfg.Inputs = []bufInput{{Directory: "api"}}
	} else {
		cfg.Inputs[0].Directory = "api"
		cfg.Inputs[0].Paths = nil
	}

	sort.Slice(cfg.Inputs, func(i, j int) bool {
		return cfg.Inputs[i].Directory < cfg.Inputs[j].Directory
	})

	out, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

type bufGenFile struct {
	Version string                   `yaml:"version"`
	Inputs  []bufInput               `yaml:"inputs"`
	Plugins []map[string]interface{} `yaml:"plugins"`
}

type bufInput struct {
	Directory string   `yaml:"directory"`
	Paths     []string `yaml:"paths,omitempty"`
}

func readModuleName(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", errors.New("module name not found in go.mod")
}

func nextMigrationName(dir string, tableName string) (string, error) {
	timestamp := time.Now().UTC().Format("20060102150405")
	name := fmt.Sprintf("%s_create_%s.sql", timestamp, tableName)
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("migration already exists: %s", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return name, nil
}

func toPascalCase(name string) string {
	parts := strings.Split(name, "_")
	var builder strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		builder.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			builder.WriteString(part[1:])
		}
	}
	return builder.String()
}

func toUpperSnake(name string) string {
	return strings.ToUpper(name)
}

func pluralize(name string) string {
	switch {
	case strings.HasSuffix(name, "y") && len(name) > 1 && !isVowel(name[len(name)-2]):
		return name[:len(name)-1] + "ies"
	case strings.HasSuffix(name, "s"),
		strings.HasSuffix(name, "x"),
		strings.HasSuffix(name, "z"),
		strings.HasSuffix(name, "ch"),
		strings.HasSuffix(name, "sh"):
		return name + "es"
	default:
		return name + "s"
	}
}

func isVowel(b byte) bool {
	switch b {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	default:
		return false
	}
}

const protoTemplate = `syntax = "proto3";

package {{.FeatureName}}.v1;

option go_package = "{{.ModuleName}}/gen/{{.FeatureName}}/v1;{{.PackageAlias}}";

import "google/api/annotations.proto";

service {{.FeatureNamePascal}}Service {
  rpc Create{{.FeatureNamePascal}}(Create{{.FeatureNamePascal}}Request) returns (Create{{.FeatureNamePascal}}Response) {
    option (google.api.http) = {
      post: "/api/v1/{{.FeatureRoutePlural}}"
      body: "*"
    };
  }

  rpc Get{{.FeatureNamePascal}}(Get{{.FeatureNamePascal}}Request) returns (Get{{.FeatureNamePascal}}Response) {
    option (google.api.http) = {
      get: "/api/v1/{{.FeatureRoutePlural}}/{id}"
    };
  }
}

message Create{{.FeatureNamePascal}}Request {
  string name = 1;
}

message Create{{.FeatureNamePascal}}Response {
  int64 id = 1;
}

message Get{{.FeatureNamePascal}}Request {
  int64 id = 1;
}

message Get{{.FeatureNamePascal}}Response {
  int64 id = 1;
}
`

const errorReasonTemplate = `syntax = "proto3";

package {{.FeatureName}}.v1;

option go_package = "{{.ModuleName}}/gen/{{.FeatureName}}/v1;{{.PackageAlias}}";

import "errors/errors.proto";

enum ErrorReason {
  option (errors.default_code) = 500;

  ERROR_REASON_UNSPECIFIED = 0;
  ERROR_REASON_{{.FeatureNameUpper}}_NOT_FOUND = 1 [(errors.code) = 404];
  ERROR_REASON_{{.FeatureNameUpper}}_ALREADY_EXISTS = 2 [(errors.code) = 409];
}
`

const portsTemplate = `package ports

import "context"

type {{.FeatureNamePascal}}Query interface {
	ExistsByID(ctx context.Context, id int64) (bool, error)
}
`

const facadeTemplate = `package {{.FeatureName}}

import "context"

// Facade implements ports.{{.FeatureNamePascal}}Query, exposing {{.FeatureName}} data to other domains.
type Facade struct {
	repo *Repo
}

func NewFacade(repo *Repo) *Facade {
	return &Facade{repo: repo}
}

func (f *Facade) ExistsByID(ctx context.Context, id int64) (bool, error) {
	_ = ctx
	_ = id
	// TODO: 补充跨域查询逻辑，例如基于 repo 查询记录是否存在。
	return false, nil
}
`

const wireBindTemplate = `package {{.FeatureName}}

import (
	"github.com/google/wire"

	"{{.ModuleName}}/internal/domain/ports"
)

// WireBind binds Facade to the ports.{{.FeatureNamePascal}}Query interface for cross-domain DI.
// Separate from wire.go so codegen won't overwrite it.
var WireBind = wire.NewSet(
	NewFacade,
	wire.Bind(new(ports.{{.FeatureNamePascal}}Query), new(*Facade)),
)
`

const schemaTemplate = `-- Schema for GORM Gen code generation (MySQL-compatible syntax)
-- Production migrations remain in db/migrations/ with PostgreSQL syntax.

CREATE TABLE {{.TableName}} (
    id         BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
`

const migrationTemplate = `-- migrate:up
CREATE TABLE {{.TableName}} (
    id         BIGSERIAL    PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- migrate:down
DROP TABLE IF EXISTS {{.TableName}};
`
