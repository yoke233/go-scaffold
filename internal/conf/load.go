package conf

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type LoadOptions struct {
	Path string
	Env  string
}

func Load(opts LoadOptions) (*Bootstrap, error) {
	if strings.TrimSpace(opts.Path) == "" {
		return nil, errors.New("config path is required")
	}

	cfg := defaultBootstrap()
	if err := mergeYAML(&cfg, opts.Path); err != nil {
		return nil, err
	}

	env := strings.TrimSpace(opts.Env)
	if env == "" {
		env = firstNonEmpty(os.Getenv("APP_ENV"), cfg.App.Env)
	}
	if env != "" {
		overridePath := envConfigPath(opts.Path, env)
		if exists(overridePath) {
			if err := mergeYAML(&cfg, overridePath); err != nil {
				return nil, err
			}
		}
		cfg.App.Env = env
	}

	applyEnvOverrides(&cfg)
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (b Bootstrap) Validate() error {
	switch {
	case strings.TrimSpace(b.App.Name) == "":
		return errors.New("app.name is required")
	case strings.TrimSpace(b.Server.HTTP.Addr) == "":
		return errors.New("server.http.addr is required")
	case strings.TrimSpace(b.Server.GRPC.Addr) == "":
		return errors.New("server.grpc.addr is required")
	case strings.TrimSpace(b.Auth.JWT.Issuer) == "":
		return errors.New("auth.jwt.issuer is required")
	case strings.TrimSpace(b.Auth.JWT.SigningKey) == "":
		return errors.New("auth.jwt.signing_key is required")
	case strings.TrimSpace(b.Auth.JWT.AccessTokenTTL) == "":
		return errors.New("auth.jwt.access_token_ttl is required")
	case strings.TrimSpace(b.Data.Database.DSN) == "":
		return errors.New("data.database.dsn is required")
	}

	if _, err := time.ParseDuration(b.Auth.JWT.AccessTokenTTL); err != nil {
		return fmt.Errorf("auth.jwt.access_token_ttl must be a valid duration: %w", err)
	}

	level := strings.ToLower(strings.TrimSpace(b.Log.Level))
	switch level {
	case "", "debug", "info", "warn", "error":
		return nil
	default:
		return fmt.Errorf("log.level must be one of debug, info, warn, error, got %q", b.Log.Level)
	}
}

func defaultBootstrap() Bootstrap {
	return Bootstrap{
		App: App{
			Name: "go-scaffold",
			Env:  "local",
		},
		Log: Log{
			Level: "info",
		},
		Auth: Auth{
			JWT: JWTAuth{
				Issuer:         "go-scaffold",
				SigningKey:     "dev-secret-change-me",
				AccessTokenTTL: "2h",
			},
		},
	}
}

func mergeYAML(cfg *Bootstrap, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}
	return nil
}

func applyEnvOverrides(cfg *Bootstrap) {
	cfg.App.Name = firstNonEmpty(os.Getenv("APP_NAME"), cfg.App.Name)
	cfg.App.Env = firstNonEmpty(os.Getenv("APP_ENV"), cfg.App.Env)
	cfg.Server.HTTP.Addr = firstNonEmpty(os.Getenv("APP_HTTP_ADDR"), cfg.Server.HTTP.Addr)
	cfg.Server.GRPC.Addr = firstNonEmpty(os.Getenv("APP_GRPC_ADDR"), cfg.Server.GRPC.Addr)
	cfg.Log.Level = firstNonEmpty(os.Getenv("APP_LOG_LEVEL"), cfg.Log.Level)
	cfg.Auth.JWT.Issuer = firstNonEmpty(os.Getenv("APP_AUTH_JWT_ISSUER"), cfg.Auth.JWT.Issuer)
	cfg.Auth.JWT.SigningKey = firstNonEmpty(os.Getenv("APP_AUTH_JWT_SIGNING_KEY"), cfg.Auth.JWT.SigningKey)
	cfg.Auth.JWT.AccessTokenTTL = firstNonEmpty(os.Getenv("APP_AUTH_JWT_ACCESS_TOKEN_TTL"), cfg.Auth.JWT.AccessTokenTTL)
	cfg.Data.Database.DSN = firstNonEmpty(os.Getenv("APP_DATABASE_DSN"), cfg.Data.Database.DSN)
}

func envConfigPath(basePath string, env string) string {
	dir := filepath.Dir(basePath)
	ext := filepath.Ext(basePath)
	name := strings.TrimSuffix(filepath.Base(basePath), ext)
	return filepath.Join(dir, name+"."+env+ext)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
