package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/faildruid/gh-sync-go/internal/config"
	"github.com/faildruid/gh-sync-go/internal/mirror"
)

// Variables for testing
var (
	mirrorAllFunc = func(cfg *config.SyncConfig) error {
		return mirror.MirrorAll(cfg)
	}
	runFunc = run
	osExit  = os.Exit
)

func run() error {
	// Create a new FlagSet for testability
	fs := flag.NewFlagSet("gh-sync", flag.ContinueOnError)
	var configPath string
	fs.StringVar(&configPath, "config", "config.yaml", "path to config.yaml")
	fs.StringVar(&configPath, "c", "config.yaml", "path to config.yaml (short)")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := mirrorAllFunc(cfg); err != nil {
		return fmt.Errorf("mirror: %w", err)
	}
	return nil
}

func main() {
	if err := runFunc(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		osExit(1)
	}
}
