package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	currentScaffoldVersion   = "v0.5.0"
	scaffoldManifestFile     = "scaffold.yaml"
	defaultOpenAPIDir        = "docs/openapi"
	defaultFeaturesGenPath   = "cmd/server/features_gen.go"
	defaultWireGenPath       = "cmd/server/wire_gen.go"
	defaultConfigPath        = "configs/config.yaml"
	defaultConfigExamplePath = "configs/config.example.yaml"
	defaultConfigLocalPath   = "configs/config.local.yaml"
	defaultConfigTestPath    = "configs/config.test.yaml"
)

type scaffoldManifest struct {
	ScaffoldVersion string `yaml:"scaffold_version"`
	Docs            struct {
		OpenAPIDir string `yaml:"openapi_dir"`
	} `yaml:"docs"`
	Generated struct {
		FeatureRegistry string `yaml:"feature_registry"`
		WireInjector    string `yaml:"wire_injector"`
	} `yaml:"generated"`
}

func defaultScaffoldManifest() scaffoldManifest {
	manifest := scaffoldManifest{
		ScaffoldVersion: currentScaffoldVersion,
	}
	manifest.Docs.OpenAPIDir = defaultOpenAPIDir
	manifest.Generated.FeatureRegistry = defaultFeaturesGenPath
	manifest.Generated.WireInjector = defaultWireGenPath
	return manifest
}

func loadScaffoldManifest(root string) (scaffoldManifest, error) {
	manifest := defaultScaffoldManifest()

	data, err := os.ReadFile(filepath.Join(root, scaffoldManifestFile))
	if err != nil {
		return manifest, err
	}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return manifest, fmt.Errorf("parse %s: %w", scaffoldManifestFile, err)
	}

	manifest.normalize()
	if strings.TrimSpace(manifest.ScaffoldVersion) == "" {
		return manifest, fmt.Errorf("parse %s: scaffold_version is required", scaffoldManifestFile)
	}

	return manifest, nil
}

func readScaffoldManifestCheck(root string) (doctorCheck, scaffoldManifest) {
	manifest, err := loadScaffoldManifest(root)
	if err == nil {
		return doctorCheck{
			Name:     scaffoldManifestFile,
			Required: true,
			OK:       true,
			Detail:   "version " + manifest.ScaffoldVersion,
		}, manifest
	}

	fallback := defaultScaffoldManifest()
	if errors.Is(err, os.ErrNotExist) {
		return doctorCheck{
			Name:     scaffoldManifestFile,
			Required: true,
			OK:       false,
			Detail:   "missing",
		}, fallback
	}

	return doctorCheck{
		Name:     scaffoldManifestFile,
		Required: true,
		OK:       false,
		Detail:   err.Error(),
	}, fallback
}

func (m *scaffoldManifest) normalize() {
	if strings.TrimSpace(m.Docs.OpenAPIDir) == "" {
		m.Docs.OpenAPIDir = defaultOpenAPIDir
	}
	if strings.TrimSpace(m.Generated.FeatureRegistry) == "" {
		m.Generated.FeatureRegistry = defaultFeaturesGenPath
	}
	if strings.TrimSpace(m.Generated.WireInjector) == "" {
		m.Generated.WireInjector = defaultWireGenPath
	}
}
