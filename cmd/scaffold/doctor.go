package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type doctorCheck struct {
	Name     string
	Required bool
	OK       bool
	Detail   string
}

func runDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	root := fs.String("root", ".", "project root")
	if err := fs.Parse(args); err != nil {
		return err
	}

	report, err := doctor(*root)
	if err != nil {
		return err
	}

	printDoctorReport(report)

	var failures int
	for _, check := range report {
		if check.Required && !check.OK {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("doctor found %d required issue(s)", failures)
	}

	return nil
}

func doctor(root string) ([]doctorCheck, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	checks := []doctorCheck{
		checkPath(absRoot, "go.mod", true),
		checkPath(absRoot, "Makefile", true),
		checkPath(absRoot, "buf.yaml", true),
		checkPath(absRoot, "buf.gen.yaml", true),
		checkPath(absRoot, filepath.Join("configs", "config.yaml"), true),
		checkPath(absRoot, filepath.Join("configs", "config.example.yaml"), true),
		checkPath(absRoot, "api", true),
		checkPath(absRoot, filepath.Join("db", "schema"), true),
		checkPath(absRoot, filepath.Join("db", "migrations"), true),
		checkTool("go", true),
		checkTool("buf", true),
		checkTool("wire", true),
		checkTool("golangci-lint", true),
		checkTool("docker", false),
	}

	return checks, nil
}

func checkPath(root string, relativePath string, required bool) doctorCheck {
	fullPath := filepath.Join(root, relativePath)
	info, err := os.Stat(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return doctorCheck{
				Name:     relativePath,
				Required: required,
				OK:       false,
				Detail:   "missing",
			}
		}
		return doctorCheck{
			Name:     relativePath,
			Required: required,
			OK:       false,
			Detail:   err.Error(),
		}
	}

	kind := "file"
	if info.IsDir() {
		kind = "dir"
	}

	return doctorCheck{
		Name:     relativePath,
		Required: required,
		OK:       true,
		Detail:   kind,
	}
}

func checkTool(name string, required bool) doctorCheck {
	path, err := exec.LookPath(name)
	if err != nil {
		return doctorCheck{
			Name:     "tool:" + name,
			Required: required,
			OK:       false,
			Detail:   "not found in PATH",
		}
	}

	return doctorCheck{
		Name:     "tool:" + name,
		Required: required,
		OK:       true,
		Detail:   path,
	}
}

func printDoctorReport(checks []doctorCheck) {
	fmt.Println("doctor report:")
	for _, check := range checks {
		status := "ok"
		if !check.OK && check.Required {
			status = "fail"
		}
		if !check.OK && !check.Required {
			status = "warn"
		}

		scope := "required"
		if !check.Required {
			scope = "optional"
		}

		fmt.Printf("  [%s] %-18s (%s) %s\n", strings.ToUpper(status), check.Name, scope, check.Detail)
	}
}
