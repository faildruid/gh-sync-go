package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigSuccess(t *testing.T) {
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
  - "fred"
work_dir: "workdir"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Source.APIURL != "https://src/api/v3" {
		t.Errorf("source api url mismatch")
	}
	if cfg.Target.APIURL != "https://dst/api/v3" {
		t.Errorf("target api url mismatch")
	}
	if len(cfg.Repos) != 1 || cfg.Repos[0] != "fred" {
		t.Errorf("repos not parsed as expected")
	}
	if filepath.Base(cfg.WorkDir) != "workdir" {
		t.Errorf("work_dir not resolved as expected")
	}
}

func TestLoadConfigMissingRepos(t *testing.T) {
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
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := Load(cfgPath); err == nil {
		t.Fatalf("expected error for missing repos, got nil")
	}
}

func TestLoadConfigMissingGitHubSections(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := `
repos:
  - "fred"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := Load(cfgPath); err == nil {
		t.Fatalf("expected error for missing github sections, got nil")
	}
}
