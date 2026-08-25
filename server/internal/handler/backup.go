package handler

import (
	"net/http"
	"strconv"

	"git-backup-web/server/internal/backup"

	"github.com/gin-gonic/gin"
)

// RunBackup 启动指定平台的一次备份任务
// 路由：POST /api/backup/run/:platform
func RunBackup(mgr *backup.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		platform := c.Param("platform")
		if !validPlatform(platform) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的平台: " + platform})
			return
		}
		id, err := mgr.Run(platform)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": id, "platform": platform})
	}
}

// CancelBackup 请求停止一条正在执行的备份任务。
// 路由：POST /api/backup/cancel/:id
func CancelBackup(mgr *backup.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil || id == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "任务 ID 无效"})
			return
		}
		if err := mgr.Cancel(uint(id)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": id, "status": "cancelling"})
	}
}
