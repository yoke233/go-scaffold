package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddFeatureCreatesProtoSkeleton(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, "go.mod"), "module example.com/project\n\ngo 1.25.6\n")
	writeFixtureFile(t, filepath.Join(root, "buf.gen.yaml"), "version: v2\ninputs:\n  - directory: .\n    paths:\n      - api/user\nplugins:\n  - local: protoc-gen-go\n    out: gen\n")

	if err := addFeature(root, "order_item"); err != nil {
		t.Fatalf("addFeature returned error: %v", err)
	}

	protoPath := filepath.Join(root, "api", "order_item", "v1", "order_item.proto")
	body := readFixtureFile(t, protoPath)
	if !strings.Contains(body, `service OrderItemService`) {
		t.Fatalf("expected generated service declaration, got:\n%s", body)
	}
	if !strings.Contains(body, `option go_package = "example.com/project/gen/order_item/v1;order_itemv1";`) {
		t.Fatalf("expected go_package declaration, got:\n%s", body)
	}

	bufGen := readFixtureFile(t, filepath.Join(root, "buf.gen.yaml"))
	if !strings.Contains(bufGen, "directory: api") {
		t.Fatalf("expected buf.gen.yaml to point at api directory, got:\n%s", bufGen)
	}
	if strings.Contains(bufGen, "paths:") {
		t.Fatalf("expected buf.gen.yaml paths to be removed, got:\n%s", bufGen)
	}

	portsBody := readFixtureFile(t, filepath.Join(root, "internal", "domain", "ports", "order_item.go"))
	if !strings.Contains(portsBody, `type OrderItemQuery interface`) {
		t.Fatalf("expected generated ports interface, got:\n%s", portsBody)
	}

	facadeBody := readFixtureFile(t, filepath.Join(root, "internal", "feature", "order_item", "facade.go"))
	if !strings.Contains(facadeBody, `type Facade struct`) {
		t.Fatalf("expected generated facade, got:\n%s", facadeBody)
	}

	wireBindBody := readFixtureFile(t, filepath.Join(root, "internal", "feature", "order_item", "wire_bind.go"))
	if !strings.Contains(wireBindBody, `ports.OrderItemQuery`) {
		t.Fatalf("expected generated wire bind, got:\n%s", wireBindBody)
	}

	schemaBody := readFixtureFile(t, filepath.Join(root, "db", "schema", "order_items.sql"))
	if !strings.Contains(schemaBody, `CREATE TABLE order_items`) {
		t.Fatalf("expected generated schema, got:\n%s", schemaBody)
	}

	migrations, err := filepath.Glob(filepath.Join(root, "db", "migrations", "*_create_order_items.sql"))
	if err != nil {
		t.Fatalf("glob migrations failed: %v", err)
	}
	if len(migrations) != 1 {
		t.Fatalf("expected one migration file, got %d", len(migrations))
	}
	migrationBody := readFixtureFile(t, migrations[0])
	if !strings.Contains(migrationBody, `DROP TABLE IF EXISTS order_items;`) {
		t.Fatalf("expected generated migration, got:\n%s", migrationBody)
	}
}

func TestAddFeatureRejectsInvalidName(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, "go.mod"), "module example.com/project\n\ngo 1.25.6\n")
	writeFixtureFile(t, filepath.Join(root, "buf.gen.yaml"), "version: v2\ninputs:\n  - directory: api\nplugins: []\n")

	if err := addFeature(root, "Order"); err == nil {
		t.Fatal("expected invalid feature name error")
	}
}

func TestAddFeatureFailsWhenFileAlreadyExists(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, "go.mod"), "module example.com/project\n\ngo 1.25.6\n")
	writeFixtureFile(t, filepath.Join(root, "buf.gen.yaml"), "version: v2\ninputs:\n  - directory: api\nplugins: []\n")
	writeFixtureFile(t, filepath.Join(root, "api", "ledger", "v1", "ledger.proto"), "syntax = \"proto3\";\n")

	if err := addFeature(root, "ledger"); err == nil {
		t.Fatal("expected duplicate file error")
	}
}

func TestPluralizeHandlesConsonantY(t *testing.T) {
	if got := pluralize("ledger_entry"); got != "ledger_entries" {
		t.Fatalf("expected ledger_entries, got %s", got)
	}
}

func writeFixtureFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
}

func readFixtureFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file failed: %v", err)
	}
	return string(data)
}
