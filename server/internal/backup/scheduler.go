package backup

import (
	"fmt"
	"log"
	"sync"
	"time"

	"git-backup-web/server/internal/db"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// Scheduler 每平台维护一份 cron。tick 周期性同步：
//  1. 删除已禁用 / 表达式变化 / Profile 关闭的旧 cron
//  2. 为新启用的平台创建新 cron
// 这样修改任意平台的定时任务都不需要重启服务。
type Scheduler struct {
	db   *gorm.DB
	mgr  *Manager
	mu   sync.Mutex
	crons map[string]*cron.Cron // key = platform
	exprs map[string]string    // key = platform，记录该平台当前的 cron 表达式
}

const syncInterval = 20 * time.Second

// NewScheduler 构造调度器
func NewScheduler(database *gorm.DB, mgr *Manager) *Scheduler {
	return &Scheduler{
		db:    database,
		mgr:   mgr,
		crons: make(map[string]*cron.Cron),
		exprs: make(map[string]string),
	}
}

// Start 在后台协程中持续同步调度
func (s *Scheduler) Start() {
	go s.loop()
}

func (s *Scheduler) loop() {
	s.sync()
	ticker := time.NewTicker(syncInterval)
	defer ticker.Stop()
	for range ticker.C {
		s.sync()
	}
}

// sync 读取最新 Profile，把每平台 cron 调整到位
func (s *Scheduler) sync() {
	s.mu.Lock()
	defer s.mu.Unlock()

	var profiles []db.Profile
	if err := s.db.Find(&profiles).Error; err != nil {
		log.Printf("[scheduler] 读取 profiles 失败: %v", err)
		return
	}

	// 当前应当存在的 (platform, expr) 集合
	desired := make(map[string]string) // platform -> expr
	for _, p := range profiles {
		if !p.Enabled || !p.ScheduleEnabled || p.ScheduleCron == "" {
			continue
		}
		desired[p.Platform] = p.ScheduleCron
	}

	// 关闭不再需要或表达式变化的 cron
	for platform, cr := range s.crons {
		if newExpr, ok := desired[platform]; !ok || newExpr != s.exprs[platform] {
			cr.Stop()
			delete(s.crons, platform)
			delete(s.exprs, platform)
			log.Printf("[scheduler] 已停止平台 %s 的定时任务", platform)
		}
	}

	// 创建或更新 cron
	for platform, expr := range desired {
		if cur, ok := s.crons[platform]; ok && s.exprs[platform] == expr {
			// 表达式未变，cron 还在跑。robfig/cron 的 AddFunc 会忽略重复 ID，
			// 这里直接跳过，避免触发 Stop / Start 之间的空窗。
			_ = cur
			continue
		}
		cr := cron.New()
		platformCopy := platform
		if _, err := cr.AddFunc(expr, func() {
			s.trigger(platformCopy)
		}); err != nil {
			log.Printf("[scheduler] 平台 %s 定时表达式无效，已跳过: %s (%v)", platformCopy, expr, err)
			continue
		}
		cr.Start()
		s.crons[platform] = cr
		s.exprs[platform] = expr
		log.Printf("[scheduler] 平台 %s 定时备份已启用: %s", platform, expr)
	}
}

// trigger 由 cron 触发一次指定平台的备份
func (s *Scheduler) trigger(platform string) {
	if _, err := s.mgr.Run(platform); err != nil {
		log.Printf("[scheduler] %s 自动备份未触发: %v", platform, err)
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	s.db.Model(&db.Profile{}).Where("platform = ?", platform).Update("schedule_last_run", now)
	log.Printf("[scheduler] %s 自动备份已触发: %s", platform, now)
}

// CronSummary 返回当前各平台的 cron 表达式（仅用于调试 / 健康检查）
func (s *Scheduler) CronSummary() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]string, len(s.exprs))
	for k, v := range s.exprs {
		out[k] = v
	}
	return out
}

var _ = fmt.Sprintf
