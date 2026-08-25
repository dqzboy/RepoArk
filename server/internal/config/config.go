package config

import (
	"encoding/json"
	"os"
)

// Config 系统配置（单例，id 固定为 1）
type Config struct {
	ID            uint   `gorm:"primaryKey" json:"-"`
	GitUser       string `json:"git_user"`
	GitToken      string `json:"git_token"`
	RepoName      string `json:"repo_name"`
	Branch        string `json:"branch"`
	BackupDir     string `json:"backup_dir"`
	ServerName    string `json:"server_name"`
	BackupSources string `json:"-"`         // 以 JSON 数组字符串存储
	HostRoot      string `json:"host_root"` // Docker 部署时宿主机根在容器内的挂载点（如 /host）
	AdminUser     string `json:"admin_user"`
	AdminPass     string `json:"admin_pass"`
	JWTSecret     string `json:"-"`

	// 定时备份
	ScheduleEnabled bool   `json:"schedule_enabled"`  // 是否开启定时备份
	ScheduleCron    string `json:"schedule_cron"`     // 标准 5 段 cron 表达式（分 时 日 月 周）
	ScheduleLastRun string `json:"schedule_last_run"` // 上次自动备份触发时间（只读展示）
}

// Default 返回初始默认配置
func Default() Config {
	sources, _ := json.Marshal([]string{"/etc/passwd", "/etc/nginx/conf.d"})
	return Config{
		GitUser:         "your_username",
		GitToken:        "",
		RepoName:        "your_repository",
		Branch:          "main",
		BackupDir:       "/app/data/tmp/github",
		ServerName:      "",
		BackupSources:   string(sources),
		AdminUser:       "admin",
		AdminPass:       "admin",
		JWTSecret:       "git-backup-change-me",
		ScheduleEnabled: false,
		ScheduleCron:    "0 2 * * *", // 默认每天凌晨 2:00
	}
}

// Sources 解析备份源路径列表
func (c Config) Sources() ([]string, error) {
	var s []string
	if c.BackupSources == "" {
		return s, nil
	}
	return s, json.Unmarshal([]byte(c.BackupSources), &s)
}

// SetSources 保存备份源路径列表
func (c *Config) SetSources(s []string) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	c.BackupSources = string(b)
	return nil
}

// EffectiveHostRoot 返回实际生效的宿主机根映射：优先使用配置值，否则回退到环境变量 HOST_ROOT
func (c Config) EffectiveHostRoot() string {
	if c.HostRoot != "" {
		return c.HostRoot
	}
	return os.Getenv("HOST_ROOT")
}
