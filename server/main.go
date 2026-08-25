package main

import (
	"log"
	"net/http"

	"git-backup-web/server/internal/backup"
	"git-backup-web/server/internal/db"
	"git-backup-web/server/internal/handler"
	"git-backup-web/server/internal/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	// 关闭 Gin 的 debug 级别日志（不再打印 [GIN-debug] 路由表与调试警告）
	gin.SetMode(gin.ReleaseMode)

	database, err := db.Init("./data/app.db")
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	mgr := backup.New(database)

	// 启动定时备份调度器（按配置中的 cron 表达式自动触发）
	scheduler := backup.NewScheduler(database, mgr)
	scheduler.Start()

	r := gin.New()
	r.Use(middleware.AccessLogger(), gin.Recovery(), corsMiddleware())

	api := r.Group("/api")
	{
		api.POST("/auth/login", handler.Login(database))
		auth := api.Group("")
		auth.Use(middleware.JWTAuth(database))
		{
			auth.GET("/platforms", handler.ListPlatforms(database))
			auth.GET("/platforms/:platform", handler.GetProfile(database))
			auth.GET("/platforms/:platform/token", handler.GetProfileToken(database))
			auth.PUT("/platforms/:platform", handler.UpdateProfile(database))
			auth.GET("/jobs", handler.ListJobs(database))
			auth.GET("/jobs/:id", handler.GetJob(database))
			auth.POST("/backup/run/:platform", handler.RunBackup(mgr))
			auth.POST("/backup/cancel/:id", handler.CancelBackup(mgr))
			auth.GET("/users", handler.ListUsers(database))
			auth.POST("/users", handler.CreateUser(database))
			auth.PUT("/users/:id", handler.UpdateUser(database))
			auth.DELETE("/users/:id", handler.DeleteUser(database))
		}
	}

	// 由 Go 直接托管内嵌的前端构建产物（SPA）：
	// 启动后端即可在 :8080 访问前端页面，无需单独启动前端服务。
	registerWebUI(r)

	log.Println("Git 备份管理服务已启动: http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
