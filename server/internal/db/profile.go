package db

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
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
	ID            uint   `gorm:"primaryKey" json:"id"`
	Platform      string `gorm:"uniqueIndex;size:16" json:"platform"` // github | gitcode | gitee
	Enabled       bool   `json:"enabled"`                             // 总开关（关闭后该平台既不手动执行也不定时执行）
	GitUser       string `json:"git_user"`
	GitToken      string `json:"-"` // 不直接序列化到前端，由后端脱敏后输出
	RepoName      string `json:"repo_name"`
	Branch        string `json:"branch"`
	BackupDir     string `json:"backup_dir"`
	ServerName    string `json:"server_name"`
	BackupSources string `json:"-"` // JSON 数组字符串存储
	HostRoot      string `json:"host_root"`

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

// DefaultBackupDir 返回平台独立的临时任务目录。
// 目录只在任务期间保存 commit/tree 元数据与新生成的 Git 对象，不保存工作区副本。
func DefaultBackupDir(platform string) string {
	return filepath.Join("/app/data/tmp", platform)
}

// ValidateNodeName 校验备份节点名称。节点名称会直接作为远程仓库中的一级目录名，
// 因此只允许单个安全路径段，避免目录穿越或意外创建多级目录。
func ValidateNodeName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("备份节点名称不能为空")
	}
	if len([]rune(name)) > 80 {
		return fmt.Errorf("备份节点名称不能超过 80 个字符")
	}
	if name == "." || name == ".." || strings.ContainsAny(name, `/\\`) {
		return fmt.Errorf("备份节点名称只能是单个目录名，不能包含路径分隔符")
	}
	if strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return fmt.Errorf("备份节点名称不能包含控制字符")
	}
	return nil
}

// NewNodeName 生成一个持久化到数据库的稳定备份节点名称。
// 可通过 REPOARK_NODE_NAME 在首次初始化时指定更易读的名称。
func NewNodeName() string {
	if configured := strings.TrimSpace(os.Getenv("REPOARK_NODE_NAME")); ValidateNodeName(configured) == nil {
		return configured
	}
	random := make([]byte, 4)
	if _, err := rand.Read(random); err == nil {
		return fmt.Sprintf("repoark-%x", random)
	}
	return fmt.Sprintf("repoark-%x", time.Now().UnixNano())
}

// CloneDefaults 用平台名构造一个空 Profile（用于首启迁移占位）
func NewEmptyProfile(platform string) *Profile {
	branch := "main"
	if platform == PlatformGitCode {
		// gitcode 默认分支常见为 master，沿用旧项目 GitHub 的 main 当占位即可，用户可在 UI 里改
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	return &Profile{
		Platform:     platform,
		Enabled:      false,
		Branch:       branch,
		BackupDir:    DefaultBackupDir(platform),
		ScheduleCron: "0 2 * * *",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
