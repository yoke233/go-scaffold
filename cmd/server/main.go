package main

import (
	"flag"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"

	"project/internal/conf"
)

func main() {
	confPath := flag.String("conf", "configs/config.yaml", "config path")
	flag.Parse()

	bc := loadConfig(*confPath)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	app, cleanup, err := wireApp(bc, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	if err := app.Run(); err != nil {
		panic(err)
	}
}

func loadConfig(path string) *conf.Bootstrap {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var bc conf.Bootstrap
	if err := yaml.Unmarshal(data, &bc); err != nil {
		panic(err)
	}
	return &bc
}
