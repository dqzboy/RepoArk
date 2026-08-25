package handler

import (
	"net/http"

	"git-backup-web/server/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// ListPlatforms 返回所有平台的 Profile 摘要（不含 Token 原文）
func ListPlatforms(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var profiles []db.Profile
		if err := database.Order("id asc").Find(&profiles).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取平台配置失败"})
			return
		}
		out := make([]gin.H, 0, len(profiles))
		for _, p := range profiles {
			sources, _ := p.Sources()
			out = append(out, profileResponse(p, sources))
		}
		c.JSON(http.StatusOK, out)
	}
}

// GetProfile 返回指定平台的 Profile 详情
func GetProfile(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		platform := c.Param("platform")
		if !validPlatform(platform) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的平台: " + platform})
			return
		}
		var p db.Profile
		if err := database.Where("platform = ?", platform).First(&p).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "平台配置不存在"})
			return
		}
		sources, _ := p.Sources()
		c.JSON(http.StatusOK, profileResponse(p, sources))
	}
}

// UpdateProfile 更新指定平台的 Profile
func UpdateProfile(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		platform := c.Param("platform")
		if !validPlatform(platform) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的平台: " + platform})
			return
		}
		var body struct {
			Enabled          bool     `json:"enabled"`
			GitUser          string   `json:"git_user"`
			GitToken         string   `json:"git_token"`
			RepoName         string   `json:"repo_name"`
			Branch           string   `json:"branch"`
			BackupDir        string   `json:"backup_dir"`
			ServerName       string   `json:"server_name"`
			BackupSources    []string `json:"backup_sources"`
			HostRoot         string   `json:"host_root"`
			ScheduleEnabled  bool     `json:"schedule_enabled"`
			ScheduleCron     string   `json:"schedule_cron"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
			return
		}
		// 定时表达式校验（仅在该平台启用定时时校验）
		if body.ScheduleEnabled && body.ScheduleCron != "" {
			if _, err := cron.ParseStandard(body.ScheduleCron); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "定时表达式格式不正确: " + err.Error()})
				return
			}
		}
		var p db.Profile
		if err := database.Where("platform = ?", platform).First(&p).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "平台配置不存在"})
			return
		}
		p.Enabled = body.Enabled
		p.GitUser = body.GitUser
		if body.GitToken != "" && body.GitToken != "********" {
			p.GitToken = body.GitToken
		}
		p.RepoName = body.RepoName
		if body.Branch != "" {
			p.Branch = body.Branch
		}
		p.BackupDir = body.BackupDir
		p.ServerName = body.ServerName
		_ = p.SetSources(body.BackupSources)
		p.HostRoot = body.HostRoot
		p.ScheduleEnabled = body.ScheduleEnabled
		p.ScheduleCron = body.ScheduleCron
		if err := database.Save(&p).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "配置已保存", "platform": platform})
	}
}

// profileResponse 把 Profile 转成前端友好的响应，Token 做脱敏
func profileResponse(p db.Profile, sources []string) gin.H {
	tokenMask := ""
	if p.GitToken != "" {
		tokenMask = "********"
	}
	return gin.H{
		"id":                p.ID,
		"platform":          p.Platform,
		"enabled":           p.Enabled,
		"git_user":          p.GitUser,
		"git_token":         tokenMask,
		"repo_name":         p.RepoName,
		"branch":            p.Branch,
		"backup_dir":        p.BackupDir,
		"server_name":       p.ServerName,
		"host_root":         p.EffectiveHostRoot(),
		"backup_sources":    sources,
		"schedule_enabled":  p.ScheduleEnabled,
		"schedule_cron":     p.ScheduleCron,
		"schedule_last_run": p.ScheduleLastRun,
	}
}

// validPlatform 校验平台合法，避免 handler 与 manager 重复实现
func validPlatform(p string) bool {
	for _, v := range db.AllPlatforms() {
		if v == p {
			return true
		}
	}
	return false
}
