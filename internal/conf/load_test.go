package conf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMergesEnvSpecificConfig(t *testing.T) {
	root := t.TempDir()
	basePath := filepath.Join(root, "config.yaml")
	writeConfigFile(t, basePath, `
app:
  name: "demo"
  env: "local"
server:
  http:
    addr: "0.0.0.0:8080"
  grpc:
    addr: "0.0.0.0:9090"
data:
  database:
    dsn: "postgres://base"
`)
	writeConfigFile(t, filepath.Join(root, "config.test.yaml"), `
server:
  http:
    addr: "127.0.0.1:18080"
log:
  level: "debug"
`)

	cfg, err := Load(LoadOptions{
		Path: basePath,
		Env:  "test",
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server.HTTP.Addr != "127.0.0.1:18080" {
		t.Fatalf("expected env override http addr, got %s", cfg.Server.HTTP.Addr)
	}
	if cfg.Server.GRPC.Addr != "0.0.0.0:9090" {
		t.Fatalf("expected base grpc addr, got %s", cfg.Server.GRPC.Addr)
	}
	if cfg.Log.Level != "debug" {
		t.Fatalf("expected debug level, got %s", cfg.Log.Level)
	}
	if cfg.App.Env != "test" {
		t.Fatalf("expected env to be test, got %s", cfg.App.Env)
	}
}

func TestLoadAppliesEnvironmentOverrides(t *testing.T) {
	root := t.TempDir()
	basePath := filepath.Join(root, "config.yaml")
	writeConfigFile(t, basePath, `
app:
  name: "demo"
server:
  http:
    addr: "0.0.0.0:8080"
  grpc:
    addr: "0.0.0.0:9090"
data:
  database:
    dsn: "postgres://base"
`)

	t.Setenv("APP_ENV", "local")
	t.Setenv("APP_HTTP_ADDR", "127.0.0.1:28080")
	t.Setenv("APP_DATABASE_DSN", "postgres://override")
	t.Setenv("APP_LOG_LEVEL", "warn")

	cfg, err := Load(LoadOptions{Path: basePath})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server.HTTP.Addr != "127.0.0.1:28080" {
		t.Fatalf("expected env http addr, got %s", cfg.Server.HTTP.Addr)
	}
	if cfg.Data.Database.DSN != "postgres://override" {
		t.Fatalf("expected env dsn override, got %s", cfg.Data.Database.DSN)
	}
	if cfg.Log.Level != "warn" {
		t.Fatalf("expected warn log level, got %s", cfg.Log.Level)
	}
}

func TestLoadValidatesRequiredFields(t *testing.T) {
	root := t.TempDir()
	basePath := filepath.Join(root, "config.yaml")
	writeConfigFile(t, basePath, `
app:
  name: ""
server:
  http:
    addr: ""
`)

	if _, err := Load(LoadOptions{Path: basePath}); err == nil {
		t.Fatal("expected validation error")
	}
}

func writeConfigFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config file failed: %v", err)
	}
}
