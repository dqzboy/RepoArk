package git

import (
	"context"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"git-backup-web/server/internal/db"
)

func TestBuildRepoURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		platform string
		owner    string
		repo     string
		want     string
	}{
		{"github", db.PlatformGitHub, "octocat", "archive", "https://github.com/octocat/archive.git"},
		{"gitcode current host", db.PlatformGitCode, "team", "archive.git", "https://gitcode.com/team/archive.git"},
		{"gitee", db.PlatformGitee, "team", "archive", "https://gitee.com/team/archive.git"},
		{"nested namespace", db.PlatformGitCode, "org/platform", "server backup", "https://gitcode.com/org/platform/server%20backup.git"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := BuildRepoURL(tt.platform, tt.owner, tt.repo)
			if err != nil {
				t.Fatalf("BuildRepoURL() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("BuildRepoURL() = %q, want %q", got, tt.want)
			}
			if strings.Contains(got, "token") || strings.Contains(got, "@") {
				t.Fatalf("repository URL must not contain credentials: %q", got)
			}
		})
	}
}

func TestBuildRepoURLRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		platform string
		owner    string
		repo     string
	}{
		{"unknown", "owner", "repo"},
		{db.PlatformGitHub, "", "repo"},
		{db.PlatformGitHub, "owner", "../repo"},
	} {
		if got, err := BuildRepoURL(tc.platform, tc.owner, tc.repo); err == nil {
			t.Fatalf("BuildRepoURL(%q, %q, %q) = %q, want error", tc.platform, tc.owner, tc.repo, got)
		}
	}
}

func TestRunBacksUpAndUpdatesForEveryPlatform(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	for _, platform := range db.AllPlatforms() {
		platform := platform
		t.Run(platform, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			remote := filepath.Join(root, "remote.git")
			seed := filepath.Join(root, "seed")
			backupDir := filepath.Join(root, "backup")
			sourceDir := filepath.Join(root, "source")
			mustRun(t, "", "init", "--bare", remote)
			mustRun(t, "", "--git-dir", remote, "config", "uploadpack.allowFilter", "true")
			mustRun(t, "", "--git-dir", remote, "config", "uploadpack.allowAnySHA1InWant", "true")
			mustRun(t, "", "init", "-b", "main", seed)
			mustRun(t, seed, "config", "user.name", "RepoArk Test")
			mustRun(t, seed, "config", "user.email", "repoark@example.test")
			mustWrite(t, filepath.Join(seed, "README.md"), "seed\n")
			mustRun(t, seed, "add", "README.md")
			mustRun(t, seed, "commit", "-m", "seed")
			mustRun(t, seed, "remote", "add", "origin", remote)
			mustRun(t, seed, "push", "-u", "origin", "main")

			mustWrite(t, filepath.Join(sourceDir, "config.txt"), "version-one\n")
			cfg := Config{
				Platform:      platform,
				GitUser:       "backup-user",
				GitToken:      "token-with-special:@value",
				RepoName:      "archive",
				Branch:        "main",
				BackupDir:     backupDir,
				ServerName:    "node-a",
				BackupSources: []string{sourceDir},
				RemoteURL:     fileURL(remote),
			}
			var logs strings.Builder
			logger := func(level, message string) {
				logs.WriteString(level + ":" + message + "\n")
			}
			if err := Run(cfg, logger); err != nil {
				t.Fatalf("first Run() error = %v\n%s", err, logs.String())
			}
			assertGitFile(t, remote, "main:node-a/source/config.txt", "version-one\n")

			mustWrite(t, filepath.Join(sourceDir, "config.txt"), "version-two\n")
			if err := Run(cfg, logger); err != nil {
				t.Fatalf("second Run() error = %v\n%s", err, logs.String())
			}
			assertGitFile(t, remote, "main:node-a/source/config.txt", "version-two\n")

			entries, err := os.ReadDir(backupDir)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".repoark-run-") {
					t.Fatalf("temporary task directory was not cleaned: %s", entry.Name())
				}
			}
			if strings.Contains(logs.String(), cfg.GitToken) {
				t.Fatal("backup logs leaked the Git token")
			}
		})
	}
}

func TestRunInitializesEmptyRemoteRepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	backupDir := filepath.Join(root, "backup")
	sourceDir := filepath.Join(root, "source")
	mustRun(t, "", "init", "--bare", remote)
	mustRun(t, "", "--git-dir", remote, "config", "uploadpack.allowFilter", "true")
	mustRun(t, "", "--git-dir", remote, "config", "uploadpack.allowAnySHA1InWant", "true")
	mustWrite(t, filepath.Join(sourceDir, "app.conf"), "enabled=true\n")

	cfg := Config{
		Platform:      db.PlatformGitHub,
		GitUser:       "backup-user",
		GitToken:      "test-token",
		RepoName:      "archive",
		Branch:        "main",
		BackupDir:     backupDir,
		ServerName:    "blog-prod",
		BackupSources: []string{sourceDir},
		RemoteURL:     fileURL(remote),
	}
	var logs strings.Builder
	if err := Run(cfg, func(level, message string) {
		logs.WriteString(level + ":" + message + "\n")
	}); err != nil {
		t.Fatalf("Run() error = %v\n%s", err, logs.String())
	}

	assertGitFile(t, remote, "main:blog-prod/source/app.conf", "enabled=true\n")
}

func TestFetchRemoteMetadataDoesNotDownloadFileBlobs(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	seed := filepath.Join(root, "seed")
	repoDir := filepath.Join(root, "client.git")
	mustRun(t, "", "init", "--bare", remote)
	mustRun(t, "", "--git-dir", remote, "config", "uploadpack.allowFilter", "true")
	mustRun(t, "", "--git-dir", remote, "config", "uploadpack.allowAnySHA1InWant", "true")
	mustRun(t, "", "init", "-b", "main", seed)
	mustRun(t, seed, "config", "user.name", "RepoArk Test")
	mustRun(t, seed, "config", "user.email", "repoark@example.test")
	mustWrite(t, filepath.Join(seed, "large.bin"), strings.Repeat("blob-content-", 8192))
	mustRun(t, seed, "add", "large.bin")
	mustRun(t, seed, "commit", "-m", "seed")
	remoteBlob := strings.TrimSpace(mustOutput(t, seed, "rev-parse", "HEAD:large.bin"))
	mustRun(t, seed, "remote", "add", "origin", fileURL(remote))
	mustRun(t, seed, "push", "origin", "main")
	mustRun(t, "", "init", "--bare", repoDir)
	mustRun(t, "", "--git-dir", repoDir, "remote", "add", "origin", fileURL(remote))

	state, err := inspectRemote(context.Background(), repoDir, "main", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !state.hasTarget || !state.supportsFilter {
		t.Fatalf("remote state = %+v, want target branch with blob filter support", state)
	}
	if err := fetchRemoteMetadata(context.Background(), repoDir, "main", nil, func(string, string) {}); err != nil {
		t.Fatal(err)
	}

	types := mustOutput(t, "", "--git-dir", repoDir, "cat-file", "--batch-check=%(objecttype)", "--batch-all-objects")
	for _, objectType := range strings.Fields(types) {
		if objectType == "blob" {
			t.Fatal("metadata-only fetch downloaded a file blob")
		}
	}

	sourceDir := filepath.Join(root, "source")
	mustWrite(t, filepath.Join(sourceDir, "new.conf"), "new=true\n")
	cfg := Config{
		Platform:      db.PlatformGitHub,
		GitUser:       "backup-user",
		GitToken:      "test-token",
		RepoName:      "archive",
		Branch:        "main",
		BackupDir:     filepath.Join(root, "tasks"),
		ServerName:    "node-a",
		BackupSources: []string{sourceDir},
		RemoteURL:     fileURL(remote),
	}
	sources, err := resolveSources(context.Background(), cfg, nil, func(string, string) {})
	if err != nil {
		t.Fatal(err)
	}
	stats, err := scanSources(context.Background(), sources, nil, func(string, string) {}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := importSnapshot(context.Background(), repoDir, cfg, sources, nil, stats, true, nil, 35, 75); err != nil {
		t.Fatal(err)
	}
	if err := runGitContext(context.Background(), "", func(string, string) {}, nil, "--git-dir", repoDir, "push", "origin", "refs/heads/main:refs/heads/main"); err != nil {
		t.Fatal(err)
	}
	objects := mustOutput(t, "", "--git-dir", repoDir, "cat-file", "--batch-check=%(objectname) %(objecttype)", "--batch-all-objects")
	if strings.Contains(objects, remoteBlob+" blob") {
		t.Fatal("creating and pushing the new commit downloaded an existing remote file blob")
	}
	assertGitFile(t, remote, "main:node-a/source/new.conf", "new=true\n")
}

func TestRunContextReportsProgressAndHonorsPreCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := RunContext(ctx, Config{}, func(string, string) {}, nil)
	if err == nil || err != context.Canceled {
		t.Fatalf("RunContext() error = %v, want context.Canceled", err)
	}

	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	sourceDir := filepath.Join(root, "source")
	mustRun(t, "", "init", "--bare", remote)
	mustWrite(t, filepath.Join(sourceDir, "app.conf"), "enabled=true\n")
	phases := map[string]bool{}
	cfg := Config{
		Platform:      db.PlatformGitHub,
		GitUser:       "backup-user",
		GitToken:      "test-token",
		RepoName:      "archive",
		Branch:        "main",
		BackupDir:     filepath.Join(root, "tasks"),
		ServerName:    "node-a",
		BackupSources: []string{sourceDir},
		RemoteURL:     fileURL(remote),
	}
	if err := RunContext(context.Background(), cfg, func(string, string) {}, func(phase string, _ int, _ string) {
		phases[phase] = true
	}); err != nil {
		t.Fatal(err)
	}
	for _, phase := range []string{"preparing", "scanning", "connecting", "packing", "committing", "pushing", "completed"} {
		if !phases[phase] {
			t.Errorf("progress did not report phase %q", phase)
		}
	}
}

func TestRunRejectsMissingSourcesAndUnsafeNodeName(t *testing.T) {
	base := Config{
		Platform:   db.PlatformGitHub,
		GitUser:    "backup-user",
		GitToken:   "test-token",
		RepoName:   "archive",
		Branch:     "main",
		BackupDir:  filepath.Join(t.TempDir(), "backup"),
		ServerName: "blog-prod",
		RemoteURL:  filepath.Join(t.TempDir(), "remote.git"),
	}
	if err := Run(base, func(string, string) {}); err == nil || !strings.Contains(err.Error(), "未配置备份源路径") {
		t.Fatalf("Run() missing sources error = %v", err)
	}

	base.BackupSources = []string{t.TempDir()}
	base.ServerName = "../escape"
	if err := Run(base, func(string, string) {}); err == nil || !strings.Contains(err.Error(), "备份节点名称") {
		t.Fatalf("Run() unsafe node error = %v", err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fileURL(path string) string {
	return (&url.URL{Scheme: "file", Path: path}).String()
}

func mustRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	_ = mustOutput(t, dir, args...)
}

func mustOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func assertGitFile(t *testing.T, remote, object, want string) {
	t.Helper()
	got := mustOutput(t, "", "--git-dir", remote, "show", object)
	if got != want {
		t.Fatalf("git show %s = %q, want %q", object, got, want)
	}
}
