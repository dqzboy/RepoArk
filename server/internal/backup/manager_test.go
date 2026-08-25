package backup

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git-backup-web/server/internal/db"
	gitbackup "git-backup-web/server/internal/git"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestRunRejectsProfileWithoutBackupSources(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "manager.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&db.Profile{}, &db.Job{}); err != nil {
		t.Fatal(err)
	}
	profile := db.Profile{
		Platform:   db.PlatformGitHub,
		Enabled:    true,
		GitUser:    "backup-user",
		GitToken:   "test-token",
		RepoName:   "archive",
		Branch:     "main",
		BackupDir:  filepath.Join(t.TempDir(), "workspace"),
		ServerName: "blog-prod",
	}
	if err := profile.SetSources(nil); err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&profile).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := New(database).Run(db.PlatformGitHub); err == nil || !strings.Contains(err.Error(), "至少配置一个备份源路径") {
		t.Fatalf("Run() error = %v", err)
	}
	var count int64
	database.Model(&db.Job{}).Count(&count)
	if count != 0 {
		t.Fatalf("job count = %d, want 0", count)
	}
}

func TestCancelStopsRunningBackup(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "manager.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&db.Profile{}, &db.Job{}); err != nil {
		t.Fatal(err)
	}
	profile := db.Profile{
		Platform:   db.PlatformGitHub,
		Enabled:    true,
		GitUser:    "backup-user",
		GitToken:   "test-token",
		RepoName:   "archive",
		Branch:     "main",
		BackupDir:  filepath.Join(t.TempDir(), "workspace"),
		ServerName: "blog-prod",
	}
	if err := profile.SetSources([]string{t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&profile).Error; err != nil {
		t.Fatal(err)
	}

	manager := New(database)
	started := make(chan struct{})
	manager.runner = func(ctx context.Context, _ gitbackup.Config, _ gitbackup.Logger, progress gitbackup.ProgressFunc) error {
		progress("packing", 47, "正在生成 Git 对象")
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}
	id, err := manager.Run(db.PlatformGitHub)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("backup runner did not start")
	}
	if err := manager.Cancel(id); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		var job db.Job
		if err := database.First(&job, id).Error; err != nil {
			t.Fatal(err)
		}
		if job.Status == db.JobStatusCancelled {
			if job.Phase != "cancelled" {
				t.Fatalf("job phase = %q, want cancelled", job.Phase)
			}
			if job.Progress != 47 {
				t.Fatalf("job progress = %d, want last reported progress 47", job.Progress)
			}
			if job.FinishedAt == "" {
				t.Fatal("cancelled job must have finished_at")
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("job status = %q, want cancelled", job.Status)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
