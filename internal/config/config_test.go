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
  - source: "fred"
    target: "barney"
  - source: "wilma"
    target: "betty"
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
	if len(cfg.Repos) != 2 {
		t.Fatalf("expected 2 repo mappings, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Source != "fred" || cfg.Repos[0].Target != "barney" {
		t.Errorf("first mapping incorrect: %+v", cfg.Repos[0])
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
  - source: "fred"
    target: "barney"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := Load(cfgPath); err == nil {
		t.Fatalf("expected error for missing github sections, got nil")
	}
}

func TestLoadConfigInvalidRepoMapping(t *testing.T) {
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
  - source: ""
    target: "barney"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := Load(cfgPath); err == nil {
		t.Fatalf("expected error for invalid repo mapping, got nil")
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	_, err := Load("nonexistent-file.yaml")
	if err == nil {
		t.Fatalf("expected error for missing file, got nil")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := `
invalid yaml content: [
  unclosed bracket
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := Load(cfgPath); err == nil {
		t.Fatalf("expected error for invalid YAML, got nil")
	}
}

func TestLoadConfigDefaultWorkDir(t *testing.T) {
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
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	// Should default to .work/repos
	if !filepath.IsAbs(cfg.WorkDir) {
		t.Errorf("work_dir should be absolute")
	}
	if filepath.Base(cfg.WorkDir) != "repos" {
		t.Errorf("expected default work_dir to end with 'repos', got %s", cfg.WorkDir)
	}
}

func TestLoadConfigMissingSourceAPIURL(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := `
github:
  source:
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
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := Load(cfgPath); err == nil {
		t.Fatalf("expected error for missing source api_url, got nil")
	}
}

func TestLoadConfigMissingTargetAPIURL(t *testing.T) {
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
    git_url: "https://dst"
    app_id: 2
    installation_id: 20
    private_key_path: "dst.pem"
    org: "org2"
repos:
  - source: "fred"
    target: "barney"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := Load(cfgPath); err == nil {
		t.Fatalf("expected error for missing target api_url, got nil")
	}
}

func TestLoadConfigRepoMissingTarget(t *testing.T) {
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
    target: ""
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := Load(cfgPath); err == nil {
		t.Fatalf("expected error for repo missing target, got nil")
	}
}
