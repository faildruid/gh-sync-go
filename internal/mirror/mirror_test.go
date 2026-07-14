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

func TestBuildGitURLWithoutScheme(t *testing.T) {
	instance := config.GitHubInstanceConfig{
		GitURL: "github.com",
		Org:    "org1",
	}
	url, err := buildGitURL(instance, "repo", "TOKEN")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "https://x-access-token:TOKEN@github.com/org1/repo.git"
	if url != expected {
		t.Fatalf("unexpected url %q, expected %q", url, expected)
	}
}

func TestBuildGitURLWithInvalidURL(t *testing.T) {
	instance := config.GitHubInstanceConfig{
		GitURL: "://invalid",
		Org:    "org1",
	}
	_, err := buildGitURL(instance, "repo", "TOKEN")
	if err == nil {
		t.Fatalf("expected error for invalid URL, got nil")
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
		Repos: []config.RepoMapping{
			{Source: "fred", Target: "barney"},
		},
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
		if len(args) >= 3 {
			_ = os.MkdirAll(args[len(args)-1], 0o755)
		}
		return nil
	}

	if err := MirrorRepo(cfg.Repos[0], cfg); err != nil {
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
		Repos: []config.RepoMapping{
			{Source: "fred", Target: "barney"},
		},
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

	if err := MirrorRepo(cfg.Repos[0], cfg); err == nil {
		t.Fatalf("expected error from MirrorRepo, got nil")
	}
}

func TestMirrorAllStopsOnError(t *testing.T) {
	origImpl := mirrorRepoImpl
	defer func() { mirrorRepoImpl = origImpl }()

	tmp := t.TempDir()
	cfg := &config.SyncConfig{
		Repos: []config.RepoMapping{
			{Source: "a", Target: "a"},
			{Source: "b", Target: "b"},
		},
		WorkDir: filepath.Join(tmp, "work"),
	}

	var called []string
	mirrorRepoImpl = func(m config.RepoMapping, cfg *config.SyncConfig) error {
		called = append(called, m.Source)
		if m.Source == "a" {
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

func TestMirrorAllSuccess(t *testing.T) {
	origImpl := mirrorRepoImpl
	defer func() { mirrorRepoImpl = origImpl }()

	tmp := t.TempDir()
	cfg := &config.SyncConfig{
		Repos: []config.RepoMapping{
			{Source: "a", Target: "a"},
			{Source: "b", Target: "b"},
		},
		WorkDir: filepath.Join(tmp, "work"),
	}

	var called []string
	mirrorRepoImpl = func(m config.RepoMapping, cfg *config.SyncConfig) error {
		called = append(called, m.Source)
		return nil
	}

	err := MirrorAll(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(called) != 2 || called[0] != "a" || called[1] != "b" {
		t.Fatalf("expected both repos to be mirrored, called=%v", called)
	}
}

func TestMirrorRepoWorkDirCreationError(t *testing.T) {
	origNew := newAppAuth
	defer func() { newAppAuth = origNew }()

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
		Repos: []config.RepoMapping{
			{Source: "fred", Target: "barney"},
		},
		WorkDir: "/dev/null/cannot-create-dir",
	}

	srcProv := &fakeProvider{token: "SRC"}
	dstProv := &fakeProvider{token: "DST"}

	newAppAuth = func(c config.GitHubInstanceConfig) tokenProvider {
		if c.Org == "org1" {
			return srcProv
		}
		return dstProv
	}

	err := MirrorRepo(cfg.Repos[0], cfg)
	if err == nil {
		t.Fatalf("expected error when work dir cannot be created, got nil")
	}
}

func TestMirrorRepoSourceTokenError(t *testing.T) {
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
		Repos: []config.RepoMapping{
			{Source: "fred", Target: "barney"},
		},
		WorkDir: filepath.Join(tmp, "work"),
	}

	dstProv := &fakeProvider{token: "DST"}

	newAppAuth = func(c config.GitHubInstanceConfig) tokenProvider {
		if c.Org == "org1" {
			return &tokenProviderWithError{}
		}
		return dstProv
	}

	runGit = func(args []string, cwd string) error {
		return nil
	}

	err := MirrorRepo(cfg.Repos[0], cfg)
	if err == nil {
		t.Fatalf("expected error from source token creation, got nil")
	}
}

func TestMirrorRepoTargetTokenError(t *testing.T) {
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
		Repos: []config.RepoMapping{
			{Source: "fred", Target: "barney"},
		},
		WorkDir: filepath.Join(tmp, "work"),
	}

	srcProv := &fakeProvider{token: "SRC"}

	newAppAuth = func(c config.GitHubInstanceConfig) tokenProvider {
		if c.Org == "org1" {
			return srcProv
		}
		return &tokenProviderWithError{}
	}

	runGit = func(args []string, cwd string) error {
		return nil
	}

	err := MirrorRepo(cfg.Repos[0], cfg)
	if err == nil {
		t.Fatalf("expected error from target token creation, got nil")
	}
}

func TestMirrorRepoSourceURLError(t *testing.T) {
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
			GitURL:         "://invalid",
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
		Repos: []config.RepoMapping{
			{Source: "fred", Target: "barney"},
		},
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
		return nil
	}

	err := MirrorRepo(cfg.Repos[0], cfg)
	if err == nil {
		t.Fatalf("expected error from invalid source URL, got nil")
	}
}

func TestMirrorRepoTargetURLError(t *testing.T) {
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
			GitURL:         "://invalid",
			AppID:          2,
			InstallationID: 20,
			PrivateKeyPath: "ignored-dst.pem",
			Org:            "org2",
		},
		Repos: []config.RepoMapping{
			{Source: "fred", Target: "barney"},
		},
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
		return nil
	}

	err := MirrorRepo(cfg.Repos[0], cfg)
	if err == nil {
		t.Fatalf("expected error from invalid target URL, got nil")
	}
}

func TestMirrorRepoPushError(t *testing.T) {
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
		Repos: []config.RepoMapping{
			{Source: "fred", Target: "barney"},
		},
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

	callCount := 0
	runGit = func(args []string, cwd string) error {
		callCount++
		// Succeed on clone, fail on push
		if callCount == 1 {
			// Simulate clone creating directory
			if len(args) >= 4 && args[0] == "clone" {
				_ = os.MkdirAll(args[3], 0o755)
			}
			return nil
		}
		return errors.New("push failed")
	}

	err := MirrorRepo(cfg.Repos[0], cfg)
	if err == nil {
		t.Fatalf("expected error from push failure, got nil")
	}
}

func TestMirrorRepoCleanupExistingDir(t *testing.T) {
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
		Repos: []config.RepoMapping{
			{Source: "fred", Target: "barney"},
		},
		WorkDir: filepath.Join(tmp, "work"),
	}

	// Pre-create the repo directory to test cleanup path
	repoDir := filepath.Join(cfg.WorkDir, "fred.git")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	// Create a file in it to verify it gets removed
	testFile := filepath.Join(repoDir, "testfile")
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
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
		if len(args) >= 4 && args[0] == "clone" {
			_ = os.MkdirAll(args[3], 0o755)
		}
		return nil
	}

	err := MirrorRepo(cfg.Repos[0], cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify the old file is gone
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Fatalf("expected test file to be removed")
	}
}

func TestRunGitImpl(t *testing.T) {
	// This is an integration test that requires git to be installed
	tmp := t.TempDir()

	// Test successful git command
	err := runGitImpl([]string{"--version"}, tmp)
	if err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Test failed git command
	err = runGitImpl([]string{"invalid-command-that-does-not-exist"}, tmp)
	if err == nil {
		t.Fatalf("expected error from invalid git command, got nil")
	}
}

type tokenProviderWithError struct{}

func (t *tokenProviderWithError) CreateInstallationToken(_ time.Time) (string, error) {
	return "", errors.New("token creation failed")
}
