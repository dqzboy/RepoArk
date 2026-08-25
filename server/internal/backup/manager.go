package backup

import (
	"fmt"
	"strings"
	"time"

	"git-backup-web/server/internal/db"
	"git-backup-web/server/internal/git"

	"gorm.io/gorm"
)

// Manager 负责创建并异步执行备份任务。
// 每平台一条 Profile；同一平台只允许一条 running 任务，不同平台可以并行。
type Manager struct {
	db *gorm.DB
}

// New 构造 Manager
func New(database *gorm.DB) *Manager {
	return &Manager{db: database}
}

// Run 触发指定平台的一次备份任务，返回任务 ID
func (m *Manager) Run(platform string) (uint, error) {
	if !validPlatform(platform) {
		return 0, fmt.Errorf("不支持的平台: %s", platform)
	}
	// 同平台并发互斥
	var running db.Job
	if err := m.db.Where("status = ? AND platform = ?", "running", platform).First(&running).Error; err == nil {
		return 0, fmt.Errorf("%s 平台已有备份任务进行中（任务 #%d）", git.PlatformDisplayName(platform), running.ID)
	}
	var p db.Profile
	if err := m.db.Where("platform = ?", platform).First(&p).Error; err != nil {
		return 0, fmt.Errorf("平台配置不存在: %s", platform)
	}
	if !p.Enabled {
		return 0, fmt.Errorf("%s 平台未启用，请先在「备份配置」中开启", git.PlatformDisplayName(platform))
	}
	if p.GitToken == "" {
		return 0, fmt.Errorf("请先在「备份配置」中填写 %s Token", git.PlatformDisplayName(platform))
	}
	if p.RepoName == "" || p.GitUser == "" {
		return 0, fmt.Errorf("请先完整配置 %s 仓库信息", git.PlatformDisplayName(platform))
	}

	serverName := p.ServerName
	if serverName == "" {
		serverName = git.DetectServerName()
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	job := db.Job{
		Platform:   platform,
		Status:     "running",
		ServerName: serverName,
		StartedAt:  now,
	}
	if err := m.db.Create(&job).Error; err != nil {
		return 0, err
	}
	go m.execute(job.ID, platform, p)
	return job.ID, nil
}

// execute 在独立 goroutine 中执行备份并更新任务记录
func (m *Manager) execute(id uint, platform string, p db.Profile) {
	var logBuf strings.Builder
	logger := func(level, msg string) {
		logBuf.WriteString(fmt.Sprintf("[%s] %s\n", level, msg))
	}
	gcfg := git.Config{
		Platform:    platform,
		GitUser:     p.GitUser,
		GitToken:    p.GitToken,
		RepoName:    p.RepoName,
		Branch:      p.Branch,
		BackupDir:   p.BackupDir,
		ServerName:  p.ServerName,
		HostRoot:    p.EffectiveHostRoot(),
	}
	if sources, err := p.Sources(); err == nil {
		gcfg.BackupSources = sources
	}
	err := git.Run(gcfg, logger)

	var job db.Job
	m.db.First(&job, id)
	job.Log = logBuf.String()
	job.FinishedAt = time.Now().Format("2006-01-02 15:04:05")
	if err != nil {
		job.Status = "failed"
		job.Message = err.Error()
	} else {
		job.Status = "success"
		job.Message = "备份完成"
	}
	m.db.Save(&job)
}

// validPlatform 校验平台合法
func validPlatform(p string) bool {
	for _, v := range db.AllPlatforms() {
		if v == p {
			return true
		}
	}
	return false
}
