package web

import (
	"database/sql"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/beyondxinxin/nixvis/internal/auth"
	"github.com/beyondxinxin/nixvis/internal/netparser"
	"github.com/beyondxinxin/nixvis/internal/stats"
	"github.com/beyondxinxin/nixvis/internal/storage"
	"github.com/beyondxinxin/nixvis/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
)

// LogFile represents a discovered log file
type LogFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// scanLogPath recursively scans a directory for Nginx access log files
func scanLogPath(path string) ([]LogFile, error) {
	var logs []LogFile

	// Get current website paths to exclude
	websiteIDs := util.GetAllWebsiteIDs()
	existingPaths := make(map[string]bool)
	for _, id := range websiteIDs {
		website, ok := util.GetWebsiteByID(id)
		if ok {
			existingPaths[website.LogPath] = true
		}
	}

	// Check if path contains wildcards
	if strings.Contains(path, "*") {
		// Handle wildcard paths
		matches, err := filepath.Glob(path)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			if !existingPaths[match] {
				logs = append(logs, LogFile{
					Name: filepath.Base(match),
					Path: match,
				})
			}
		}
		return logs, nil
	}

	// Check if it's a single file
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		// Single file
		if !existingPaths[path] && isAccessLog(path) {
			logs = append(logs, LogFile{
				Name: filepath.Base(path),
				Path: path,
			})
		}
		return logs, nil
	}

	// It's a directory, walk it recursively
	err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if it's an access log
		if isAccessLog(filePath) && !existingPaths[filePath] {
			logs = append(logs, LogFile{
				Name: filepath.Base(filePath),
				Path: filePath,
			})
		}

		return nil
	})

	return logs, err
}

// isAccessLog checks if a file is an Nginx access log
func isAccessLog(path string) bool {
	base := filepath.Base(path)
	lower := strings.ToLower(base)

	// Match patterns: *access.log, *-access.log, *.log
	return strings.Contains(lower, "access.log") ||
		strings.HasSuffix(lower, "-access.log") ||
		(strings.HasSuffix(lower, ".log") && !strings.Contains(lower, "error"))
}

func SetupRoutes(
	router *gin.Engine,
	statsFactory *stats.StatsFactory,
	repo *storage.Repository,
	logParser *storage.LogParser,
	db *sql.DB) {

	// 加载模板
	tmpl, err := LoadTemplates()
	if err != nil {
		logrus.Fatalf("无法加载模板: %v", err)
	}
	router.SetHTMLTemplate(tmpl)

	// 设置静态文件服务
	staticFS, err := GetStaticFS()
	if err != nil {
		logrus.Fatalf("无法加载静态文件: %v", err)
	}

	// Initialize authentication system
	userStore := auth.NewSQLiteUserStore(db)
	if err := userStore.InitSchema(); err != nil {
		logrus.WithError(err).Error("Failed to initialize user schema")
	}

	// Generate or load JWT secret
	secretKey := "nixvis-jwt-secret-key-change-in-production"
	// In production, you should load this from a secure config or environment variable

	authManager := auth.NewManager(secretKey, userStore)

	router.StaticFS("/static", staticFS)

	// Public routes
	public := router.Group("/")
	{
		public.GET("/login", func(c *gin.Context) {
			c.HTML(http.StatusOK, "login.html", gin.H{
				"title": "NixVis - 登录",
			})
		})
		public.GET("/favicon.ico", func(c *gin.Context) {
			data, err := fs.ReadFile(staticFiles, "assets/static/favicon.ico")
			if err != nil {
				c.Status(http.StatusNotFound)
				return
			}
			c.Data(http.StatusOK, "image/x-icon", data)
		})

		// Public API routes (with optional auth)
		public.GET("/api/websites", func(c *gin.Context) {
			websiteIDs := util.GetAllWebsiteIDs()

			websites := make([]map[string]string, 0, len(websiteIDs))
			for _, id := range websiteIDs {
				website, ok := util.GetWebsiteByID(id)
				if !ok {
					continue
				}

				websites = append(websites, map[string]string{
					"id":   id,
					"name": website.Name,
				})
			}

			c.JSON(http.StatusOK, gin.H{
				"websites": websites,
			})
		})

		public.GET("/api/stats/:type", func(c *gin.Context) {
			statsType := c.Param("type")
			params := make(map[string]string)
			for key, values := range c.Request.URL.Query() {
				if len(values) > 0 {
					params[key] = values[0]
				}
			}

			query, err := statsFactory.BuildQueryFromRequest(statsType, params)

			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			// 执行查询
			result, err := statsFactory.QueryStats(statsType, query)
			if err != nil {
				logrus.WithError(err).Errorf("查询统计数据[%s]失败", statsType)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": fmt.Sprintf("查询失败: %v", err),
				})
				return
			}

			c.JSON(http.StatusOK, result)
		})
	}

	// Authentication API routes
	authAPI := router.Group("/api/auth")
	{
		// Check if system is initialized
		authAPI.GET("/check", func(c *gin.Context) {
			initialized, err := authManager.IsInitialized()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"initialized": initialized,
			})
		})

		// Initialize first admin user
		authAPI.POST("/initialize", func(c *gin.Context) {
			var req auth.InitializeUserRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
				return
			}

			if len(req.Password) < 6 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "密码至少需要6个字符"})
				return
			}

			if err := authManager.InitializeUser(req.Username, req.Password); err != nil {
				if err == auth.ErrUserAlreadyExists {
					c.JSON(http.StatusConflict, gin.H{"error": "系统已经初始化"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "管理员账户创建成功，请登录",
			})
		})

		// Login
		authAPI.POST("/login", func(c *gin.Context) {
			var req auth.LoginRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
				return
			}

			response, err := authManager.Login(req.Username, req.Password)
			if err != nil {
				if err == auth.ErrInvalidCredentials {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// Set token in cookie
			c.SetSameSite(http.SameSiteStrictMode)
			c.SetCookie(
				"auth_token",
				response.Token,
				int(24*time.Hour.Seconds()),
				"/",
				"",
				false,
				true,
			)

			c.JSON(http.StatusOK, response)
		})

		// Logout
		authAPI.POST("/logout", func(c *gin.Context) {
			// Get token from header or cookie
			token := c.GetHeader("Authorization")
			if token != "" && strings.HasPrefix(token, "Bearer ") {
				token = strings.TrimPrefix(token, "Bearer ")
			} else {
				token, _ = c.Cookie("auth_token")
			}

			authManager.Logout(token)

			// Clear cookie
			c.SetSameSite(http.SameSiteStrictMode)
			c.SetCookie(
				"auth_token",
				"",
				-1,
				"/",
				"",
				false,
				true,
			)

			c.JSON(http.StatusOK, gin.H{"success": true})
		})

		// Get current user info
		authAPI.GET("/me", authManager.AuthMiddleware(), func(c *gin.Context) {
			username, _ := auth.GetUsername(c)
			c.JSON(http.StatusOK, gin.H{
				"username": username,
				"authenticated": true,
			})
		})

		// Change password (requires authentication)
		authAPI.POST("/change-password", authManager.AuthMiddleware(), func(c *gin.Context) {
			userID, exists := auth.GetUserID(c)
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
				return
			}

			var req auth.ChangePasswordRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
				return
			}

			if len(req.NewPassword) < 6 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "新密码至少需要6个字符"})
				return
			}

			// Get user by ID to verify old password
			user, err := userStore.GetUserByID(userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "用户不存在"})
				return
			}

			// Verify old password
			if !auth.CheckPassword(req.OldPassword, user.PasswordHash) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "原密码错误"})
				return
			}

			// Update password
			newHash, err := auth.HashPassword(req.NewPassword)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
				return
			}

			if err := userStore.UpdatePassword(userID, newHash); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "密码更新失败"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "密码修改成功",
			})
		})
	}

	// Protected routes (require authentication)
	protected := router.Group("/")
	protected.Use(authManager.AuthMiddleware())
	{
		protected.GET("/", func(c *gin.Context) {
			c.HTML(http.StatusOK, "index.html", gin.H{
				"title": "NixVis - Nginx访问统计",
			})
		})
		protected.GET("/logs", func(c *gin.Context) {
			c.HTML(http.StatusOK, "logs.html", gin.H{
				"title": "NixVis - 访问日志查看",
			})
		})
		protected.GET("/spiders", func(c *gin.Context) {
			c.HTML(http.StatusOK, "spiders.html", gin.H{
				"title": "NixVis - 蜘蛛访问统计",
			})
		})
		protected.GET("/suspicious", func(c *gin.Context) {
			c.HTML(http.StatusOK, "suspicious.html", gin.H{
				"title": "NixVis - 可疑IP管理",
			})
		})
		protected.GET("/settings", func(c *gin.Context) {
			c.HTML(http.StatusOK, "settings.html", gin.H{
				"title": "NixVis - 站点设置",
			})
		})
	}

	// Protected API routes
	protectedAPI := router.Group("/api")
	protectedAPI.Use(authManager.AuthMiddleware())
	{
		// Spider stats
		protectedAPI.GET("/spiders", func(c *gin.Context) {
			websiteID := c.Query("id")
			if websiteID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "缺少网站 ID"})
				return
			}

			timeRange := time.Now().AddDate(0, 0, -7).Unix()
			spiderStats, err := repo.GetSpiderStats(websiteID, timeRange)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"spiders": spiderStats,
			})
		})

		// Suspicious IPs
		protectedAPI.GET("/suspicious", func(c *gin.Context) {
			websiteID := c.Query("id")
			if websiteID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "缺少网站 ID"})
				return
			}

			suspiciousIPs, err := repo.GetSuspiciousIPs(websiteID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"suspicious_ips": suspiciousIPs,
			})
		})

		// Block IP
		protectedAPI.POST("/block", func(c *gin.Context) {
			var req struct {
				IP        string `json:"ip"`
				WebsiteID string `json:"website_id"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
				return
			}

			if err := repo.BlockIP(req.WebsiteID, req.IP); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			blockResult := netparser.BlockIP(req.IP)
			c.JSON(http.StatusOK, blockResult)
		})

		// ========== Settings API ==========

		// GET /api/settings - 获取当前配置
		protectedAPI.GET("/settings", func(c *gin.Context) {
			websiteIDs := util.GetAllWebsiteIDs()

			websites := make([]map[string]string, 0, len(websiteIDs))
			for _, id := range websiteIDs {
				website, ok := util.GetWebsiteByID(id)
				if !ok {
					continue
				}

				websites = append(websites, map[string]string{
					"id":      id,
					"name":    website.Name,
					"logPath": website.LogPath,
				})
			}

			// 获取PV过滤配置
			pvFilter := util.GetPVFilterConfig()

			c.JSON(http.StatusOK, gin.H{
				"websites":        websites,
				"excludePatterns": pvFilter.ExcludePatterns,
				"excludeIPs":      pvFilter.ExcludeIPs,
			})
		})

		// POST /api/settings/scan - 扫描日志路径
		protectedAPI.POST("/settings/scan", func(c *gin.Context) {
			var req struct {
				Path string `json:"path"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
				return
			}

			if req.Path == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "路径不能为空"})
				return
			}

			logs, err := scanLogPath(req.Path)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"logs": logs,
			})
		})

		// POST /api/settings/add - 添加站点
		protectedAPI.POST("/settings/add", func(c *gin.Context) {
			var req struct {
				Name    string `json:"name"`
				LogPath string `json:"logPath"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
				return
			}

			if req.Name == "" || req.LogPath == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "站点名称和日志路径不能为空"})
				return
			}

			if err := util.AddWebsite(req.Name, req.LogPath); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 重新加载配置
			if err := util.ReloadConfig(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "配置保存成功，但重新加载失败: " + err.Error()})
				return
			}

			// 为新站点创建数据库表
			id := util.GenerateID(req.Name)
			if err := repo.CreateTableForWebsite(id); err != nil {
				logrus.WithError(err).Errorf("创建站点 %s 的数据库表失败", req.Name)
			}

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"id":      id,
			})
		})

		// DELETE /api/settings/remove/:id - 删除站点
		protectedAPI.DELETE("/settings/remove/:id", func(c *gin.Context) {
			id := c.Param("id")
			if id == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "缺少站点 ID"})
				return
			}

			if err := util.RemoveWebsite(id); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"success": true,
			})
		})

		// POST /api/settings/reload - 重新加载配置
		protectedAPI.POST("/settings/reload", func(c *gin.Context) {
			if err := util.ReloadConfig(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "配置已重新加载",
			})
		})

		// POST /api/settings/scan-logs - 立即触发日志扫描
		protectedAPI.POST("/settings/scan-logs", func(c *gin.Context) {
			// 重新加载配置以获取最新的网站列表
			if err := util.ReloadConfig(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "重新加载配置失败: " + err.Error()})
				return
			}

			// 执行日志扫描
			results := logParser.ScanNginxLogs()

			// 统计结果
			totalEntries := 0
			successCount := 0
			scanResults := make([]map[string]interface{}, 0)

			for _, result := range results {
				if result.WebName == "" {
					continue
				}

				totalEntries += result.TotalEntries

				scanResults = append(scanResults, map[string]interface{}{
					"name":     result.WebName,
					"id":       result.WebID,
					"success":  result.Success,
					"entries":  result.TotalEntries,
					"duration": result.Duration.Seconds(),
					"error":    "",
				})

				if result.Success {
					successCount++
				} else if result.Error != nil {
					scanResults[len(scanResults)-1]["error"] = result.Error.Error()
				}
			}

			c.JSON(http.StatusOK, gin.H{
				"success":       true,
				"total_entries": totalEntries,
				"scanned":       successCount,
				"results":       scanResults,
				"message":       fmt.Sprintf("扫描完成: %d 个站点，共 %d 条记录", successCount, totalEntries),
			})
		})

		// POST /api/settings/exclude-patterns - 添加排除模式
		protectedAPI.POST("/settings/exclude-patterns", func(c *gin.Context) {
			var req struct {
				Pattern string `json:"pattern"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
				return
			}

			if req.Pattern == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "排除模式不能为空"})
				return
			}

			if err := util.AddExcludePattern(req.Pattern); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			util.ResetConfigCache()
			netparser.InitPVFilters()

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"pattern": req.Pattern,
			})
		})

		// DELETE /api/settings/exclude-patterns/:pattern - 删除排除模式
		protectedAPI.DELETE("/settings/exclude-patterns/:pattern", func(c *gin.Context) {
			pattern := c.Param("pattern")
			if pattern == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "缺少排除模式"})
				return
			}

			if err := util.RemoveExcludePattern(pattern); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			util.ResetConfigCache()
			netparser.InitPVFilters()

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"pattern": pattern,
			})
		})

		// POST /api/settings/exclude-ips - 添加排除IP
		protectedAPI.POST("/settings/exclude-ips", func(c *gin.Context) {
			var req struct {
				IP string `json:"ip"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
				return
			}

			if req.IP == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "排除IP不能为空"})
				return
			}

			if err := util.AddExcludeIP(req.IP); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			util.ResetConfigCache()
			netparser.InitPVFilters()

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"ip":      req.IP,
			})
		})

		// DELETE /api/settings/exclude-ips/:ip - 删除排除IP
		protectedAPI.DELETE("/settings/exclude-ips/:ip", func(c *gin.Context) {
			ip := c.Param("ip")
			if ip == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "缺少排除IP"})
				return
			}

			if err := util.RemoveExcludeIP(ip); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			util.ResetConfigCache()
			netparser.InitPVFilters()

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"ip":      ip,
			})
		})
	}
}
