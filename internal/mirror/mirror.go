package mirror

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/faildruid/gh-sync-go/internal/config"
	"github.com/faildruid/gh-sync-go/internal/githubapp"
)

type tokenProvider interface {
	CreateInstallationToken(now time.Time) (string, error)
}

var newAppAuth = func(cfg config.GitHubInstanceConfig) tokenProvider {
	return &githubapp.AppAuth{
		APIURL:         cfg.APIURL,
		AppID:          cfg.AppID,
		InstallationID: cfg.InstallationID,
		PrivateKeyPath: cfg.PrivateKeyPath,
	}
}

var runGit = runGitImpl

func buildGitURL(instance config.GitHubInstanceConfig, repo string, accessToken string) (string, error) {
	parsed, err := url.Parse(instance.GitURL)
	if err != nil {
		return "", fmt.Errorf("parse git_url %s: %w", instance.GitURL, err)
	}
	scheme := parsed.Scheme
	if scheme == "" {
		scheme = "https"
	}
	host := parsed.Host
	if host == "" {
		host = parsed.Path
	}
	return fmt.Sprintf("%s://x-access-token:%s@%s/%s/%s.git",
		scheme, accessToken, host, instance.Org, repo), nil
}

func runGitImpl(args []string, cwd string) error {
	cmd := exec.Command("git", args...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %v failed: %w", args, err)
	}
	return nil
}

var MirrorRepo = func(repo string, cfg *config.SyncConfig) error {
	if err := os.MkdirAll(cfg.WorkDir, 0o755); err != nil {
		return fmt.Errorf("create work_dir: %w", err)
	}
	repoDir := filepath.Join(cfg.WorkDir, repo+".git")

	if _, err := os.Stat(repoDir); err == nil {
		if err := os.RemoveAll(repoDir); err != nil {
			return fmt.Errorf("cleanup previous repo dir: %w", err)
		}
	}

	sourceAuth := newAppAuth(cfg.Source)
	targetAuth := newAppAuth(cfg.Target)

	now := time.Now()

	sourceToken, err := sourceAuth.CreateInstallationToken(now)
	if err != nil {
		return fmt.Errorf("create source installation token: %w", err)
	}
	targetToken, err := targetAuth.CreateInstallationToken(now)
	if err != nil {
		return fmt.Errorf("create target installation token: %w", err)
	}

	sourceURL, err := buildGitURL(cfg.Source, repo, sourceToken)
	if err != nil {
		return err
	}
	targetURL, err := buildGitURL(cfg.Target, repo, targetToken)
	if err != nil {
		return err
	}

	if err := runGit([]string{"clone", "--bare", sourceURL, repoDir}, ""); err != nil {
		return err
	}
	if err := runGit([]string{"push", "--mirror", targetURL}, repoDir); err != nil {
		return err
	}

	return nil
}

func MirrorAll(cfg *config.SyncConfig) error {
	for _, repo := range cfg.Repos {
		if err := MirrorRepo(repo, cfg); err != nil {
			return err
		}
	}
	return nil
}
