package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestUpgradeCheckPassesWhenVersionMatches(t *testing.T) {
	root := t.TempDir()
	writeScaffoldManifestFixture(t, root, currentScaffoldVersion)
	writeFixtureFile(t, filepath.Join(root, defaultFeaturesGenPath), "package main\n")
	writeFixtureFile(t, filepath.Join(root, defaultWireGenPath), "package main\n")
	mustMkdir(t, filepath.Join(root, defaultOpenAPIDir))

	checks, err := upgradeCheck(root)
	if err != nil {
		t.Fatalf("upgradeCheck returned error: %v", err)
	}

	for _, check := range checks {
		if check.Required && !check.OK {
			t.Fatalf("expected required check %s to pass, got detail %s", check.Name, check.Detail)
		}
	}
}

func TestUpgradeCheckFailsWhenVersionMismatched(t *testing.T) {
	root := t.TempDir()
	writeScaffoldManifestFixture(t, root, "v0.3.0")
	writeFixtureFile(t, filepath.Join(root, defaultFeaturesGenPath), "package main\n")
	writeFixtureFile(t, filepath.Join(root, defaultWireGenPath), "package main\n")

	checks, err := upgradeCheck(root)
	if err != nil {
		t.Fatalf("upgradeCheck returned error: %v", err)
	}

	var found bool
	for _, check := range checks {
		if check.Name == "scaffold_version" {
			found = true
			if check.OK {
				t.Fatal("expected scaffold version mismatch to fail")
			}
			if !strings.Contains(check.Detail, "project=v0.3.0") {
				t.Fatalf("unexpected detail: %s", check.Detail)
			}
		}
	}
	if !found {
		t.Fatal("expected scaffold_version check to exist")
	}
}
