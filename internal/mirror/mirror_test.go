package mirror

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/faildruid/gh-sync-go/internal/config"
)

func TestBuildGitURL(t *testing.T) {
	instance := config.GitHubInstanceConfig{
		APIURL: "https://src/api/v3",
		GitURL: "https://src",
		Org:    "org1",
	}
	url, err := buildGitURL(instance, "fred", "TOKEN")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "https://x-access-token:TOKEN@src/org1/fred.git"
	if url != expected {
		t.Fatalf("unexpected url %q, expected %q", url, expected)
	}
}

type fakeProvider struct {
	token  string
	called bool
}

func (f *fakeProvider) CreateInstallationToken(_ time.Time) (string, error) {
	f.called = true
	return f.token, nil
}

func TestMirrorRepoUsesRunGitAndTokens(t *testing.T) {
	origNew := newAppAuth
	origRun := runGit
	defer func() {
		newAppAuth = origNew
		runGit = origRun
	}()

	tmp := t.TempDir()
	cfg := &config.SyncConfig{
		Source: config.GitHubInstanceConfig{
			APIURL:         "https://src/api/v3",
			GitURL:         "https://src",
			AppID:          1,
			InstallationID: 10,
			PrivateKeyPath: "ignored-src.pem",
			Org:            "org1",
		},
		Target: config.GitHubInstanceConfig{
			APIURL:         "https://dst/api/v3",
			GitURL:         "https://dst",
			AppID:          2,
			InstallationID: 20,
			PrivateKeyPath: "ignored-dst.pem",
			Org:            "org2",
		},
		Repos:   []string{"fred"},
		WorkDir: filepath.Join(tmp, "work"),
	}

	srcProv := &fakeProvider{token: "SRC"}
	dstProv := &fakeProvider{token: "DST"}

	newAppAuth = func(c config.GitHubInstanceConfig) tokenProvider {
		if c.Org == "org1" {
			return srcProv
		}
		return dstProv
	}

	type call struct {
		args []string
		cwd  string
	}
	var calls []call

	runGit = func(args []string, cwd string) error {
		cp := make([]string, len(args))
		copy(cp, args)
		calls = append(calls, call{args: cp, cwd: cwd})
		// Simulate git clone --bare creating the repo directory
		if len(args) >= 4 && args[0] == "clone" && args[1] == "--bare" {
			if err := os.MkdirAll(args[3], 0o755); err != nil {
				return err
			}
		}
		return nil
	}

	if err := MirrorRepo("fred", cfg); err != nil {
		t.Fatalf("MirrorRepo returned error: %v", err)
	}

	if !srcProv.called || !dstProv.called {
		t.Fatalf("expected both token providers to be called")
	}

	if len(calls) != 2 {
		t.Fatalf("expected 2 git calls, got %d", len(calls))
	}

	cloneCall := calls[0]
	pushCall := calls[1]

	if len(cloneCall.args) < 4 || cloneCall.args[0] != "clone" || cloneCall.args[1] != "--bare" {
		t.Fatalf("unexpected clone args: %v", cloneCall.args)
	}
	expectedRepoDir := filepath.Join(cfg.WorkDir, "fred.git")
	if cloneCall.args[3] != expectedRepoDir {
		t.Fatalf("unexpected clone target %q, expected %q", cloneCall.args[3], expectedRepoDir)
	}
	if cloneCall.cwd != "" {
		t.Fatalf("expected empty cwd for clone, got %q", cloneCall.cwd)
	}

	if len(pushCall.args) < 2 || pushCall.args[0] != "push" || pushCall.args[1] != "--mirror" {
		t.Fatalf("unexpected push args: %v", pushCall.args)
	}
	if pushCall.cwd != expectedRepoDir {
		t.Fatalf("expected cwd %q for push, got %q", expectedRepoDir, pushCall.cwd)
	}

	if _, err := os.Stat(expectedRepoDir); err != nil {
		t.Fatalf("expected repo dir to exist: %v", err)
	}
}

func TestMirrorRepoPropagatesRunGitError(t *testing.T) {
	origNew := newAppAuth
	origRun := runGit
	defer func() {
		newAppAuth = origNew
		runGit = origRun
	}()

	tmp := t.TempDir()
	cfg := &config.SyncConfig{
		Source: config.GitHubInstanceConfig{
			APIURL:         "https://src/api/v3",
			GitURL:         "https://src",
			AppID:          1,
			InstallationID: 10,
			PrivateKeyPath: "ignored-src.pem",
			Org:            "org1",
		},
		Target: config.GitHubInstanceConfig{
			APIURL:         "https://dst/api/v3",
			GitURL:         "https://dst",
			AppID:          2,
			InstallationID: 20,
			PrivateKeyPath: "ignored-dst.pem",
			Org:            "org2",
		},
		Repos:   []string{"fred"},
		WorkDir: filepath.Join(tmp, "work"),
	}

	srcProv := &fakeProvider{token: "SRC"}
	dstProv := &fakeProvider{token: "DST"}

	newAppAuth = func(c config.GitHubInstanceConfig) tokenProvider {
		if c.Org == "org1" {
			return srcProv
		}
		return dstProv
	}

	runGit = func(args []string, cwd string) error {
		return errors.New("git failure")
	}

	if err := MirrorRepo("fred", cfg); err == nil {
		t.Fatalf("expected error from MirrorRepo, got nil")
	}
}

func TestMirrorAllStopsOnError(t *testing.T) {
	origMirror := MirrorRepo
	defer func() { MirrorRepo = origMirror }()

	tmp := t.TempDir()
	cfg := &config.SyncConfig{
		Repos:   []string{"a", "b"},
		WorkDir: filepath.Join(tmp, "work"),
	}

	var called []string
	MirrorRepo = func(repo string, cfg *config.SyncConfig) error {
		called = append(called, repo)
		if repo == "a" {
			return errors.New("fail")
		}
		return nil
	}

	err := MirrorAll(cfg)
	if err == nil {
		t.Fatalf("expected error from MirrorAll, got nil")
	}
	if len(called) != 1 || called[0] != "a" {
		t.Fatalf("expected MirrorAll to stop after first error, called=%v", called)
	}
}
