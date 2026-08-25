package db

import (
	"os"
	"path/filepath"
	"time"

	"git-backup-web/server/internal/config"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Job 一次备份任务的记录
type Job struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	Platform   string `gorm:"size:16;index" json:"platform"` // github | gitcode | gitee（用于筛选展示）
	Status     string `json:"status"`                       // running | success | failed
	ServerName string `json:"server_name"`
	Message    string `json:"message"`
	Log        string `json:"log"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at"`
}

// User 后台用户（密码以 bcrypt 哈希存储）
type User struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Username  string `gorm:"uniqueIndex;size:64" json:"username"`
	Password  string `gorm:"size:128" json:"-"`
	Role      string `gorm:"size:16" json:"role"` // admin | viewer
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Init 初始化 SQLite 数据库，自动建表、写入默认配置并播种管理员账号。
// 还会把旧 Config 表里的 GitHub 字段自动迁移到 profiles 表（一条 GitHub Profile）。
func Init(path string) (*gorm.DB, error) {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	database, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	if err := database.AutoMigrate(&config.Config{}, &Job{}, &User{}, &Profile{}); err != nil {
		return nil, err
	}
	var c config.Config
	if err := database.First(&c).Error; err != nil {
		def := config.Default()
		if err := database.Create(&def).Error; err != nil {
			return nil, err
		}
		c = def
	}
	// 首次启动：按配置中的管理员账号播种一个用户
	var ucount int64
	database.Model(&User{}).Count(&ucount)
	if ucount == 0 {
		hash, herr := bcrypt.GenerateFromPassword([]byte(c.AdminPass), 10)
		if herr == nil {
			now := time.Now().Format("2006-01-02 15:04:05")
			database.Create(&User{
				Username:  c.AdminUser,
				Password:  string(hash),
				Role:      "admin",
				CreatedAt: now,
				UpdatedAt: now,
			})
		}
	}
	// 首启迁移：profiles 表为空时，自动创建 3 个平台的占位 Profile。
	// 若旧 Config 中有 GitHub 凭证（Token/Repo 非默认值），把其克隆到 GitHub Profile。
	migrateProfiles(database, c)
	return database, nil
}

// migrateProfiles 一次性把旧 Config 表的 GitHub 配置克隆为 GitHub Profile，
// 并为其它两个平台创建默认占位记录（未启用）。
func migrateProfiles(database *gorm.DB, c config.Config) {
	var count int64
	database.Model(&Profile{}).Count(&count)
	if count > 0 {
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")

	// GitHub：尝试复用旧 config 中的字段
	gh := NewEmptyProfile(PlatformGitHub)
	gh.Enabled = c.ScheduleEnabled
	gh.GitUser = c.GitUser
	gh.GitToken = c.GitToken
	gh.RepoName = c.RepoName
	gh.Branch = c.Branch
	gh.BackupDir = c.BackupDir
	gh.ServerName = c.ServerName
	gh.HostRoot = c.HostRoot
	gh.ScheduleEnabled = c.ScheduleEnabled
	gh.ScheduleCron = c.ScheduleCron
	gh.ScheduleLastRun = c.ScheduleLastRun
	gh.UpdatedAt = now
	if sources, err := c.Sources(); err == nil {
		_ = gh.SetSources(sources)
	} else {
		gh.BackupSources = c.BackupSources // 兜底原样写入
	}
	database.Create(gh)

	// GitCode / Gitee 占位
	for _, p := range []string{PlatformGitCode, PlatformGitee} {
		profile := NewEmptyProfile(p)
		profile.UpdatedAt = now
		database.Create(profile)
	}
}
