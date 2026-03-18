package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func runUpgrade(args []string) error {
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	root := fs.String("root", ".", "project root")
	checkOnly := fs.Bool("check", false, "check scaffold version and generated artifacts")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if !*checkOnly {
		return errors.New("upgrade currently supports only --check")
	}

	report, err := upgradeCheck(*root)
	if err != nil {
		return err
	}

	printUpgradeReport(report)

	var failures int
	for _, check := range report {
		if check.Required && !check.OK {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("upgrade check found %d required issue(s)", failures)
	}

	return nil
}

func upgradeCheck(root string) ([]doctorCheck, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	manifest, err := loadScaffoldManifest(absRoot)
	if err != nil {
		return nil, err
	}

	checks := []doctorCheck{
		{
			Name:     "scaffold_version",
			Required: true,
			OK:       manifest.ScaffoldVersion == currentScaffoldVersion,
			Detail:   fmt.Sprintf("project=%s tool=%s", manifest.ScaffoldVersion, currentScaffoldVersion),
		},
		checkPathWithHint(absRoot, manifest.Generated.FeatureRegistry, true, "run make generate"),
		checkPathWithHint(absRoot, manifest.Generated.WireInjector, true, "run make generate"),
		checkPathWithHint(absRoot, manifest.Docs.OpenAPIDir, false, "run make docs"),
	}

	return checks, nil
}
