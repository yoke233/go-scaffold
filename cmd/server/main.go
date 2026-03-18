package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"project/internal/conf"
)

func main() {
	confPath := flag.String("conf", "configs/config.yaml", "config path")
	appEnv := flag.String("env", "", "app env, for example local/test/prod")
	flag.Parse()

	bc := loadConfig(*confPath, *appEnv)
	logger := newLogger(bc)

	app, cleanup, err := wireApp(bc, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	if err := app.Run(); err != nil {
		panic(err)
	}
}

func loadConfig(path string, env string) *conf.Bootstrap {
	bc, err := conf.Load(conf.LoadOptions{
		Path: path,
		Env:  env,
	})
	if err != nil {
		panic(fmt.Errorf("load config: %w", err))
	}

	return bc
}

func newLogger(bc *conf.Bootstrap) *slog.Logger {
	var level slog.Level
	switch bc.Log.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}
