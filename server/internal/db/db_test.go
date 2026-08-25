package db

import (
	"path/filepath"
	"testing"
)

func TestInitCreatesIndependentTemporaryDirectoriesAndStableNodeName(t *testing.T) {
	t.Setenv("REPOARK_NODE_NAME", "blog-prod")
	databasePath := filepath.Join(t.TempDir(), "data", "app.db")
	database, err := Init(databasePath)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	var profiles []Profile
	if err := database.Order("id asc").Find(&profiles).Error; err != nil {
		t.Fatal(err)
	}
	if len(profiles) != len(AllPlatforms()) {
		t.Fatalf("profile count = %d, want %d", len(profiles), len(AllPlatforms()))
	}
	seenDirs := map[string]bool{}
	for _, profile := range profiles {
		if profile.BackupDir != DefaultBackupDir(profile.Platform) {
			t.Errorf("%s backup dir = %q, want %q", profile.Platform, profile.BackupDir, DefaultBackupDir(profile.Platform))
		}
		if seenDirs[profile.BackupDir] {
			t.Errorf("temporary directory is shared by multiple platforms: %q", profile.BackupDir)
		}
		seenDirs[profile.BackupDir] = true
		if profile.ServerName != "blog-prod" {
			t.Errorf("%s node name = %q, want blog-prod", profile.Platform, profile.ServerName)
		}
	}

	// 再次初始化同一数据库时，节点名称应保持不变。
	t.Setenv("REPOARK_NODE_NAME", "changed-env-name")
	database, err = Init(databasePath)
	if err != nil {
		t.Fatalf("second Init() error = %v", err)
	}
	var profile Profile
	if err := database.Where("platform = ?", PlatformGitHub).First(&profile).Error; err != nil {
		t.Fatal(err)
	}
	if profile.ServerName != "blog-prod" {
		t.Fatalf("persisted node name = %q, want blog-prod", profile.ServerName)
	}
}

func TestMigrateLegacySharedBackupDirectory(t *testing.T) {
	database, err := Init(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := database.Model(&Profile{}).Where("1 = 1").Update("backup_dir", "/data/backup").Error; err != nil {
		t.Fatal(err)
	}

	migrateProfileBackupDirs(database)
	var profiles []Profile
	if err := database.Find(&profiles).Error; err != nil {
		t.Fatal(err)
	}
	for _, profile := range profiles {
		if profile.BackupDir != DefaultBackupDir(profile.Platform) {
			t.Errorf("%s backup dir = %q, want %q", profile.Platform, profile.BackupDir, DefaultBackupDir(profile.Platform))
		}
	}
}

func TestMigrateLegacyPlatformRepositoryDirectories(t *testing.T) {
	database, err := Init(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	for _, platform := range AllPlatforms() {
		legacy := filepath.Join("/app/data/repos", platform)
		if err := database.Model(&Profile{}).Where("platform = ?", platform).Update("backup_dir", legacy).Error; err != nil {
			t.Fatal(err)
		}
	}

	migrateProfileBackupDirs(database)
	var profiles []Profile
	if err := database.Find(&profiles).Error; err != nil {
		t.Fatal(err)
	}
	for _, profile := range profiles {
		if profile.BackupDir != DefaultBackupDir(profile.Platform) {
			t.Errorf("%s backup dir = %q, want %q", profile.Platform, profile.BackupDir, DefaultBackupDir(profile.Platform))
		}
	}
}

func TestValidateNodeName(t *testing.T) {
	for _, valid := range []string{"blog-prod", "北京节点", "node_01.example"} {
		if err := ValidateNodeName(valid); err != nil {
			t.Errorf("ValidateNodeName(%q) error = %v", valid, err)
		}
	}
	for _, invalid := range []string{"", ".", "..", "../escape", `a\\b`, "line\nbreak"} {
		if err := ValidateNodeName(invalid); err == nil {
			t.Errorf("ValidateNodeName(%q) = nil, want error", invalid)
		}
	}
}
