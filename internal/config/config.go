package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type GitHubInstanceConfig struct {
	APIURL         string `yaml:"api_url"`
	GitURL         string `yaml:"git_url"`
	AppID          int64  `yaml:"app_id"`
	InstallationID int64  `yaml:"installation_id"`
	PrivateKeyPath string `yaml:"private_key_path"`
	Org            string `yaml:"org"`
}

type SyncConfig struct {
	Source  GitHubInstanceConfig `yaml:"source"`
	Target  GitHubInstanceConfig `yaml:"target"`
	Repos   []string             `yaml:"repos"`
	WorkDir string               `yaml:"work_dir"`
}

type rootConfig struct {
	GitHub struct {
		Source GitHubInstanceConfig `yaml:"source"`
		Target GitHubInstanceConfig `yaml:"target"`
	} `yaml:"github"`
	Repos   []string `yaml:"repos"`
	WorkDir string   `yaml:"work_dir"`
}

func Load(path string) (*SyncConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var rc rootConfig
	if err := yaml.Unmarshal(data, &rc); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if rc.GitHub.Source.APIURL == "" || rc.GitHub.Target.APIURL == "" {
		return nil, fmt.Errorf("github.source and github.target must be set")
	}

	if len(rc.Repos) == 0 {
		return nil, fmt.Errorf("repos list must not be empty")
	}

	workDir := rc.WorkDir
	if workDir == "" {
		workDir = ".work/repos"
	}

	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, fmt.Errorf("resolve work_dir: %w", err)
	}

	return &SyncConfig{
		Source:  rc.GitHub.Source,
		Target:  rc.GitHub.Target,
		Repos:   rc.Repos,
		WorkDir: absWorkDir,
	}, nil
}
