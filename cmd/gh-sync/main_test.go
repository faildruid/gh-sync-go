package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/faildruid/gh-sync-go/internal/config"
)

func TestRunSuccess(t *testing.T) {
	// Save and restore original args and working directory
	oldArgs := os.Args
	oldMirrorAll := mirrorAllFunc
	defer func() {
		os.Args = oldArgs
		mirrorAllFunc = oldMirrorAll
	}()

	// Create a temporary config file
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := `
github:
  source:
    api_url: "https://src/api/v3"
    git_url: "https://src"
    app_id: 1
    installation_id: 10
    private_key_path: "src.pem"
    org: "org1"
  target:
    api_url: "https://dst/api/v3"
    git_url: "https://dst"
    app_id: 2
    installation_id: 20
    private_key_path: "dst.pem"
    org: "org2"
repos:
  - source: "fred"
    target: "barney"
work_dir: "` + filepath.Join(dir, "work") + `"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Mock mirrorAll to succeed
	mirrorCalled := false
	mirrorAllFunc = func(cfg *config.SyncConfig) error {
		mirrorCalled = true
		return nil
	}

	// Set args to use our test config
	os.Args = []string{"gh-sync", "-config", cfgPath}

	err := run()
	if err != nil {
		t.Fatalf("run() returned error: %v", err)
	}

	if !mirrorCalled {
		t.Fatalf("expected mirrorAll to be called")
	}
}

func TestRunConfigLoadError(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use a non-existent config file
	os.Args = []string{"gh-sync", "-config", "nonexistent.yaml"}

	err := run()
	if err == nil {
		t.Fatalf("expected error for missing config file, got nil")
	}
}

func TestRunMirrorError(t *testing.T) {
	oldArgs := os.Args
	oldMirrorAll := mirrorAllFunc
	defer func() {
		os.Args = oldArgs
		mirrorAllFunc = oldMirrorAll
	}()

	// Create a temporary config file
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := `
github:
  source:
    api_url: "https://src/api/v3"
    git_url: "https://src"
    app_id: 1
    installation_id: 10
    private_key_path: "src.pem"
    org: "org1"
  target:
    api_url: "https://dst/api/v3"
    git_url: "https://dst"
    app_id: 2
    installation_id: 20
    private_key_path: "dst.pem"
    org: "org2"
repos:
  - source: "fred"
    target: "barney"
work_dir: "` + filepath.Join(dir, "work") + `"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Mock mirrorAll to fail
	mirrorAllFunc = func(cfg *config.SyncConfig) error {
		return os.ErrPermission
	}

	os.Args = []string{"gh-sync", "-config", cfgPath}

	err := run()
	if err == nil {
		t.Fatalf("expected error from mirrorAll, got nil")
	}
}

func TestRunShortFlag(t *testing.T) {
	oldArgs := os.Args
	oldMirrorAll := mirrorAllFunc
	defer func() {
		os.Args = oldArgs
		mirrorAllFunc = oldMirrorAll
	}()

	// Create a temporary config file
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := `
github:
  source:
    api_url: "https://src/api/v3"
    git_url: "https://src"
    app_id: 1
    installation_id: 10
    private_key_path: "src.pem"
    org: "org1"
  target:
    api_url: "https://dst/api/v3"
    git_url: "https://dst"
    app_id: 2
    installation_id: 20
    private_key_path: "dst.pem"
    org: "org2"
repos:
  - source: "fred"
    target: "barney"
work_dir: "` + filepath.Join(dir, "work") + `"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Mock mirrorAll to succeed
	mirrorCalled := false
	mirrorAllFunc = func(cfg *config.SyncConfig) error {
		mirrorCalled = true
		return nil
	}

	// Test the short flag version
	os.Args = []string{"gh-sync", "-c", cfgPath}

	err := run()
	if err != nil {
		t.Fatalf("run() with -c flag returned error: %v", err)
	}

	if !mirrorCalled {
		t.Fatalf("expected mirrorAll to be called")
	}
}

func TestMain(t *testing.T) {
	// This tests the main function by mocking run
	oldRun := runFunc
	oldExit := osExit
	defer func() {
		runFunc = oldRun
		osExit = oldExit
	}()

	// Test successful run
	exitCode := -1
	osExit = func(code int) {
		exitCode = code
	}
	runFunc = func() error {
		return nil
	}

	main()
	if exitCode != -1 {
		t.Fatalf("expected no exit call on success, got exit code %d", exitCode)
	}
}

func TestMainWithError(t *testing.T) {
	// Test main function with error
	oldRun := runFunc
	oldExit := osExit
	defer func() {
		runFunc = oldRun
		osExit = oldExit
	}()

	exitCode := -1
	osExit = func(code int) {
		exitCode = code
	}
	runFunc = func() error {
		return os.ErrPermission
	}

	main()
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 on error, got %d", exitCode)
	}
}

func TestRunInvalidFlag(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use an invalid flag
	os.Args = []string{"gh-sync", "-invalid-flag"}

	err := run()
	if err == nil {
		t.Fatalf("expected error for invalid flag, got nil")
	}
}

func TestRunDefaultConfig(t *testing.T) {
	oldArgs := os.Args
	oldMirrorAll := mirrorAllFunc
	defer func() {
		os.Args = oldArgs
		mirrorAllFunc = oldMirrorAll
	}()

	// Create config.yaml in current directory (the default)
	content := `
github:
  source:
    api_url: "https://src/api/v3"
    git_url: "https://src"
    app_id: 1
    installation_id: 10
    private_key_path: "src.pem"
    org: "org1"
  target:
    api_url: "https://dst/api/v3"
    git_url: "https://dst"
    app_id: 2
    installation_id: 20
    private_key_path: "dst.pem"
    org: "org2"
repos:
  - source: "fred"
    target: "barney"
`
	tmpFile := "config.yaml"
	if err := os.WriteFile(tmpFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	defer os.Remove(tmpFile)

	// Mock mirrorAll to succeed
	mirrorCalled := false
	mirrorAllFunc = func(cfg *config.SyncConfig) error {
		mirrorCalled = true
		return nil
	}

	// Don't pass any flags - should use default config.yaml
	os.Args = []string{"gh-sync"}

	err := run()
	if err != nil {
		t.Fatalf("run() with default config returned error: %v", err)
	}

	if !mirrorCalled {
		t.Fatalf("expected mirrorAll to be called")
	}
}
