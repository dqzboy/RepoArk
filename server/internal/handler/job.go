package handler

import (
	"net/http"
	"strconv"

	"git-backup-web/server/internal/db"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListJobs 返回任务历史（按 ID 倒序，可选 ?platform=github 过滤）
func ListJobs(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := database.Order("id desc").Limit(100)
		if platform := c.Query("platform"); platform != "" {
			q = q.Where("platform = ?", platform)
		}
		var jobs []db.Job
		q.Find(&jobs)
		c.JSON(http.StatusOK, jobs)
	}
}

// GetJob 返回单条任务详情（含日志）
func GetJob(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务 ID"})
			return
		}
		var job db.Job
		if err := database.First(&job, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
			return
		}
		c.JSON(http.StatusOK, job)
	}
}
