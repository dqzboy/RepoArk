package db

import (
	"encoding/json"
	"os"
	"time"
)

// Platform 平台枚举
const (
	PlatformGitHub  = "github"
	PlatformGitCode = "gitcode"
	PlatformGitee   = "gitee"
)

// AllPlatforms 返回系统支持的全部平台（用于前端 Tabs 与后端调度遍历）
func AllPlatforms() []string {
	return []string{PlatformGitHub, PlatformGitCode, PlatformGitee}
}

// Profile 一个平台账号下的备份配置 + 定时任务。
// 当前模型：一个平台一条记录，平台字段 unique 保证不会重复创建。
type Profile struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Platform    string `gorm:"uniqueIndex;size:16" json:"platform"` // github | gitcode | gitee
	Enabled     bool   `json:"enabled"`                            // 总开关（关闭后该平台既不手动执行也不定时执行）
	GitUser     string `json:"git_user"`
	GitToken    string `json:"-"` // 不直接序列化到前端，由后端脱敏后输出
	RepoName    string `json:"repo_name"`
	Branch      string `json:"branch"`
	BackupDir   string `json:"backup_dir"`
	ServerName  string `json:"server_name"`
	BackupSources string `json:"-"` // JSON 数组字符串存储
	HostRoot    string `json:"host_root"`

	// 定时备份（按平台独立）
	ScheduleEnabled bool   `json:"schedule_enabled"`
	ScheduleCron    string `json:"schedule_cron"`
	ScheduleLastRun string `json:"schedule_last_run"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Sources 解析备份源路径列表
func (p Profile) Sources() ([]string, error) {
	var s []string
	if p.BackupSources == "" {
		return s, nil
	}
	return s, json.Unmarshal([]byte(p.BackupSources), &s)
}

// SetSources 保存备份源路径列表
func (p *Profile) SetSources(s []string) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	p.BackupSources = string(b)
	return nil
}

// EffectiveHostRoot 返回实际生效的宿主机根映射：优先使用配置值，否则回退到环境变量 HOST_ROOT
func (p Profile) EffectiveHostRoot() string {
	if p.HostRoot != "" {
		return p.HostRoot
	}
	return os.Getenv("HOST_ROOT")
}

// CloneDefaults 用平台名构造一个空 Profile（用于首启迁移占位）
func NewEmptyProfile(platform string) *Profile {
	branch := "main"
	backupDir := "/data/backup"
	if platform == PlatformGitCode {
		// gitcode 默认分支常见为 master，沿用旧项目 GitHub 的 main 当占位即可，用户可在 UI 里改
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	return &Profile{
		Platform:         platform,
		Enabled:          false,
		Branch:           branch,
		BackupDir:        backupDir,
		ScheduleCron:     "0 2 * * *",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}
