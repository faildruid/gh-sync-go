package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/faildruid/gh-sync-go/internal/config"
	"github.com/faildruid/gh-sync-go/internal/mirror"
)

func run() error {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "path to config.yaml")
	flag.StringVar(&configPath, "c", "config.yaml", "path to config.yaml (short)")
	flag.Parse()

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := mirror.MirrorAll(cfg); err != nil {
		return fmt.Errorf("mirror: %w", err)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(1)
	}
}
