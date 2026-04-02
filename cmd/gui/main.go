package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/yourorg/toolkit/pkg/config"
	"github.com/yourorg/toolkit/pkg/core"
)

type Server struct {
	manager   core.PluginManager
	logger    hclog.Logger
	staticDir string
	apiKeys   map[string]bool
}

func main() {
	cfg, err := config.LoadConfig("tools.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "toolkit-gui",
		Level: hclog.Info,
	})

	manager := core.NewPluginManager(&core.ManagerConfig{
		PluginDirs: cfg.PluginDirs,
	}, logger)

	if err := manager.LoadPlugins(); err != nil {
		logger.Error("failed to load plugins", "error", err)
	}

	configs := make(map[string]map[string]interface{})
	for name, pCfg := range cfg.Plugins {
		if pCfg.Enabled {
			configs[name] = pCfg.Config
		}
	}
	if err := manager.InitializeAll(configs); err != nil {
		logger.Warn("some plugins failed to initialize", "error", err)
	}

	defer manager.ShutdownAll()

	// 获取静态文件目录的绝对路径
	staticDir, err := filepath.Abs("cmd/gui/static")
	if err != nil {
		logger.Error("failed to get static directory", "error", err)
		os.Exit(1)
	}

	// 初始化 API 密钥 - 生产环境中应从配置文件或环境变量加载
	apiKeys := make(map[string]bool)
	apiKeys["your-secret-api-key"] = true

	server := &Server{
		manager:   manager,
		logger:    logger,
		staticDir: staticDir,
		apiKeys:   apiKeys,
	}

	mux := http.NewServeMux()
	server.setupRoutes(mux)

	httpServer := &http.Server{
		Addr:    ":8082",
		Handler: securityHeaders(mux),
	}

	go func() {
		logger.Info("Starting server", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("Server exited properly")
}

func (s *Server) setupRoutes(mux *http.ServeMux) {
	auth := &AuthMiddleware{apiKeys: s.apiKeys}

	// API 路由 - 需要认证
	mux.HandleFunc("GET /api/plugins", auth.Check(func(w http.ResponseWriter, r *http.Request) {
		plugins := s.manager.ListPlugins()
		response := make([]*PluginResponse, 0, len(plugins))

		for _, mp := range plugins {
			meta := mp.Tool.Metadata()
			response = append(response, &PluginResponse{
				Name:        mp.Info.Name,
				Version:     meta.Version,
				Description: meta.Description,
				Author:      meta.Author,
				Category:    meta.Category,
				Tags:        meta.Tags,
				Enabled:     mp.Info.Enabled,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))

	mux.HandleFunc("GET /api/plugins/{name}", auth.Check(func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")

		sanitizedName, err := sanitizePluginName(name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		mp, err := s.manager.GetPlugin(sanitizedName)
		if err != nil {
			http.Error(w, "Plugin not found", http.StatusNotFound)
			return
		}

		meta := mp.Tool.Metadata()
		response := &PluginResponse{
			Name:        mp.Info.Name,
			Version:     meta.Version,
			Description: meta.Description,
			Author:      meta.Author,
			Category:    meta.Category,
			Tags:        meta.Tags,
			Enabled:     mp.Info.Enabled,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))

	mux.HandleFunc("POST /api/execute", auth.Check(func(w http.ResponseWriter, r *http.Request) {
		var req ExecuteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		sanitizedName, err := sanitizePluginName(req.Plugin)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		result, err := s.manager.ExecutePlugin(sanitizedName, req.Params)
		if err != nil {
			response := ExecuteResponse{
				Success: false,
				Error:   err.Error(),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		response := ExecuteResponse{
			Success: result.Success,
			Data:    result.Data,
			Error:   result.Error,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))

	mux.HandleFunc("POST /api/remove", auth.Check(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		sanitizedName, err := sanitizePluginName(req.Name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 关闭插件
		if err := s.manager.ShutdownPlugin(sanitizedName); err != nil {
			response := map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// 注销插件
		if err := s.manager.UnregisterPlugin(sanitizedName); err != nil {
			response := map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success": true,
			"message": "插件删除成功",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))

	// 导入工具 API
	mux.HandleFunc("POST /api/import", auth.Check(func(w http.ResponseWriter, r *http.Request) {
		// 处理FormData类型的请求
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			// 忽略错误，因为这可能是JSON类型的请求
		}
		// 简单实现：仅记录导入请求
		response := map[string]interface{}{
			"success": true,
			"message": "工具导入成功（演示模式）",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))

	// 健康检查
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"status":  "healthy",
			"plugins": len(s.manager.ListPlugins()),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// 处理静态文件
	fileServer := http.FileServer(http.Dir(s.staticDir))
	mux.Handle("/", http.StripPrefix("/", fileServer))
}

type PluginResponse struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	Enabled     bool     `json:"enabled"`
}

type ExecuteRequest struct {
	Plugin string                 `json:"plugin"`
	Params map[string]interface{} `json:"params"`
}

type ExecuteResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type AuthMiddleware struct {
	apiKeys map[string]bool
}

func (a *AuthMiddleware) Check(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			next(w, r)
			return
		}
		key := r.Header.Get("X-API-Key")
		if key == "" || !a.apiKeys[key] {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// securityHeaders 添加安全响应头
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS 头信息
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// 防止 XSS 攻击
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		// 防止 MIME 类型嗅探
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// 防止点击劫持
		w.Header().Set("X-Frame-Options", "DENY")
		// 内容安全策略
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self';")
		// Referrer 策略
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// 特性策略
		w.Header().Set("Permissions-Policy", "geolocation=(), camera=(), microphone=()")
		next.ServeHTTP(w, r)
	})
}

func sanitizePluginName(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("plugin name cannot be empty")
	}

	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(name) {
		return "", fmt.Errorf("invalid plugin name: %s", name)
	}

	return name, nil
}
