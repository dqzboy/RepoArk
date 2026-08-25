package handler

import (
	"net/http"

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
