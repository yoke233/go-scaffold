package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDoctorPassesWithRequiredFilesAndTools(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, "go.mod"), "module example.com/project\n")
	writeFixtureFile(t, filepath.Join(root, "Makefile"), "help:\n")
	writeFixtureFile(t, filepath.Join(root, "buf.yaml"), "version: v2\n")
	writeFixtureFile(t, filepath.Join(root, "buf.gen.yaml"), "version: v2\n")
	writeFixtureFile(t, filepath.Join(root, "configs", "config.yaml"), "app:\n  name: demo\n")
	writeFixtureFile(t, filepath.Join(root, "configs", "config.example.yaml"), "app:\n  name: demo\n")
	mustMkdir(t, filepath.Join(root, "api"))
	mustMkdir(t, filepath.Join(root, "db", "schema"))
	mustMkdir(t, filepath.Join(root, "db", "migrations"))

	toolDir := t.TempDir()
	for _, name := range []string{"go", "buf", "wire", "golangci-lint", "docker"} {
		createFakeTool(t, toolDir, name)
	}
	setToolPath(t, toolDir)

	checks, err := doctor(root)
	if err != nil {
		t.Fatalf("doctor returned error: %v", err)
	}

	for _, check := range checks {
		if check.Required && !check.OK {
			t.Fatalf("expected required check %s to pass, got detail %s", check.Name, check.Detail)
		}
	}
}

func TestDoctorFailsWhenRequiredToolMissing(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, "go.mod"), "module example.com/project\n")
	writeFixtureFile(t, filepath.Join(root, "Makefile"), "help:\n")
	writeFixtureFile(t, filepath.Join(root, "buf.yaml"), "version: v2\n")
	writeFixtureFile(t, filepath.Join(root, "buf.gen.yaml"), "version: v2\n")
	writeFixtureFile(t, filepath.Join(root, "configs", "config.yaml"), "app:\n  name: demo\n")
	writeFixtureFile(t, filepath.Join(root, "configs", "config.example.yaml"), "app:\n  name: demo\n")
	mustMkdir(t, filepath.Join(root, "api"))
	mustMkdir(t, filepath.Join(root, "db", "schema"))
	mustMkdir(t, filepath.Join(root, "db", "migrations"))

	toolDir := t.TempDir()
	for _, name := range []string{"go", "buf", "wire"} {
		createFakeTool(t, toolDir, name)
	}
	setToolPath(t, toolDir)

	checks, err := doctor(root)
	if err != nil {
		t.Fatalf("doctor returned error: %v", err)
	}

	var found bool
	for _, check := range checks {
		if check.Name == "tool:golangci-lint" {
			found = true
			if check.OK {
				t.Fatal("expected missing golangci-lint to fail")
			}
		}
	}
	if !found {
		t.Fatal("expected golangci-lint check to exist")
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
}

func createFakeTool(t *testing.T, dir string, name string) {
	t.Helper()

	fileName := name
	content := "#!/bin/sh\nexit 0\n"
	if runtime.GOOS == "windows" {
		fileName = name + ".cmd"
		content = "@echo off\r\nexit /b 0\r\n"
	}

	path := filepath.Join(dir, fileName)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake tool failed: %v", err)
	}
}

func setToolPath(t *testing.T, toolDir string) {
	t.Helper()

	t.Setenv("PATH", toolDir)
	if runtime.GOOS == "windows" {
		t.Setenv("PATHEXT", strings.Join([]string{".CMD", ".EXE", ".BAT"}, ";"))
	}
}
