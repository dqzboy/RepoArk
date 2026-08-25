package git

import (
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
				RemoteURL:     remote,
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

			origin := strings.TrimSpace(mustOutput(t, backupDir, "remote", "get-url", "origin"))
			if origin != remote {
				t.Fatalf("origin URL = %q, want clean URL %q", origin, remote)
			}
			if strings.Contains(logs.String(), cfg.GitToken) {
				t.Fatal("backup logs leaked the Git token")
			}
		})
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
