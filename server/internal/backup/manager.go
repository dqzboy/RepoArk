package backup

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"git-backup-web/server/internal/db"
	"git-backup-web/server/internal/git"

	"gorm.io/gorm"
)

// Manager 负责创建并异步执行备份任务。
// 每平台一条 Profile；同一平台只允许一条 running 任务，不同平台可以并行。
type Manager struct {
	db      *gorm.DB
	mu      sync.Mutex
	cancels map[uint]context.CancelFunc
	runner  func(context.Context, git.Config, git.Logger, git.ProgressFunc) error
}

// New 构造 Manager
func New(database *gorm.DB) *Manager {
	m := &Manager{
		db:      database,
		cancels: make(map[uint]context.CancelFunc),
		runner:  git.RunContext,
	}
	// 进程重启后旧 goroutine 已不存在，不能让历史任务永久停留在“进行中”。
	now := time.Now().Format("2006-01-02 15:04:05")
	database.Model(&db.Job{}).
		Where("status IN ?", []string{db.JobStatusRunning, db.JobStatusCancelling}).
		Updates(map[string]any{
			"status":      db.JobStatusFailed,
			"phase":       "failed",
			"message":     "服务重启，备份任务已中断",
			"finished_at": now,
		})
	return m
}

// Run 触发指定平台的一次备份任务，返回任务 ID
func (m *Manager) Run(platform string) (uint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !validPlatform(platform) {
		return 0, fmt.Errorf("不支持的平台: %s", platform)
	}
	// 同平台并发互斥
	var running db.Job
	if err := m.db.Where("status IN ? AND platform = ?", []string{db.JobStatusRunning, db.JobStatusCancelling}, platform).First(&running).Error; err == nil {
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
	sources, err := p.Sources()
	if err != nil {
		return 0, fmt.Errorf("备份源路径配置格式错误: %w", err)
	}
	hasSource := false
	for _, source := range sources {
		if strings.TrimSpace(source) != "" {
			hasSource = true
			break
		}
	}
	if !hasSource {
		return 0, fmt.Errorf("请至少配置一个备份源路径；临时任务目录不是备份源路径")
	}

	if err := db.ValidateNodeName(p.ServerName); err != nil {
		return 0, err
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	job := db.Job{
		Platform:   platform,
		Status:     db.JobStatusRunning,
		Phase:      "preparing",
		Progress:   1,
		ServerName: p.ServerName,
		Message:    "任务已创建，正在准备备份",
		StartedAt:  now,
	}
	if err := m.db.Create(&job).Error; err != nil {
		return 0, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancels[job.ID] = cancel
	go m.execute(ctx, job.ID, platform, p)
	return job.ID, nil
}

// Cancel 请求取消一条正在执行的备份任务。
func (m *Manager) Cancel(id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var job db.Job
	if err := m.db.First(&job, id).Error; err != nil {
		return fmt.Errorf("任务不存在")
	}
	if job.Status == db.JobStatusCancelling {
		return nil
	}
	if job.Status != db.JobStatusRunning {
		return fmt.Errorf("任务 #%d 当前状态为 %s，无法取消", id, job.Status)
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	cancel, ok := m.cancels[id]
	if !ok {
		// 理论上只会发生在进程异常恢复后；避免遗留无法结束的 running 记录。
		return m.db.Model(&db.Job{}).Where("id = ? AND status = ?", id, db.JobStatusRunning).Updates(map[string]any{
			"status":      db.JobStatusCancelled,
			"phase":       "cancelled",
			"message":     "备份任务已取消",
			"finished_at": now,
		}).Error
	}
	if err := m.db.Model(&db.Job{}).Where("id = ? AND status = ?", id, db.JobStatusRunning).Updates(map[string]any{
		"status":  db.JobStatusCancelling,
		"phase":   "cancelling",
		"message": "正在停止当前备份操作…",
	}).Error; err != nil {
		return err
	}
	cancel()
	return nil
}

// execute 在独立 goroutine 中执行备份并更新任务记录
func (m *Manager) execute(ctx context.Context, id uint, platform string, p db.Profile) {
	defer func() {
		m.mu.Lock()
		delete(m.cancels, id)
		m.mu.Unlock()
	}()

	var logBuf strings.Builder
	lastLogPersist := time.Time{}
	lastProgress := 1
	logger := func(level, msg string) {
		logBuf.WriteString(fmt.Sprintf("[%s] %s\n", level, msg))
		if time.Since(lastLogPersist) >= 500*time.Millisecond {
			m.db.Model(&db.Job{}).Where("id = ?", id).Update("log", logBuf.String())
			lastLogPersist = time.Now()
		}
	}
	progress := func(phase string, percent int, message string) {
		if percent < 0 {
			percent = 0
		} else if percent > 100 {
			percent = 100
		}
		lastProgress = percent
		m.db.Model(&db.Job{}).
			Where("id = ? AND status = ?", id, db.JobStatusRunning).
			Updates(map[string]any{
				"phase":    phase,
				"progress": percent,
				"message":  message,
				"log":      logBuf.String(),
			})
		lastLogPersist = time.Now()
	}
	gcfg := git.Config{
		Platform:   platform,
		GitUser:    p.GitUser,
		GitToken:   p.GitToken,
		RepoName:   p.RepoName,
		Branch:     p.Branch,
		BackupDir:  p.BackupDir,
		ServerName: p.ServerName,
		HostRoot:   p.EffectiveHostRoot(),
	}
	gcfg.BackupSources, _ = p.Sources()
	err := m.runner(ctx, gcfg, logger, progress)

	status := db.JobStatusSuccess
	phase := "completed"
	percent := 100
	message := "备份完成"
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		logger("INFO", "备份任务已由用户取消")
		status = db.JobStatusCancelled
		phase = "cancelled"
		percent = lastProgress
		message = "备份任务已取消"
	} else if err != nil {
		status = db.JobStatusFailed
		phase = "failed"
		percent = lastProgress
		message = err.Error()
	}
	m.db.Model(&db.Job{}).Where("id = ?", id).Updates(map[string]any{
		"status":      status,
		"phase":       phase,
		"progress":    percent,
		"message":     message,
		"log":         logBuf.String(),
		"finished_at": time.Now().Format("2006-01-02 15:04:05"),
	})
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
