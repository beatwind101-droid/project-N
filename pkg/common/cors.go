package common

import (
	"os"
	"strings"
)

// IsValidOrigin 验证源是否允许
// 支持：
// 1. 从环境变量 ALLOWED_ORIGIN 加载允许的源列表
// 2. 支持逗号分隔的多个源
// 3. 支持通配符域名（如 *.example.com）
// 4. 开发环境默认允许 http://localhost:8082 和 http://127.0.0.1:8082
func IsValidOrigin(origin string) bool {
	// 从环境变量获取允许的源列表
	allowedOrigins := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigins == "" {
		// 开发环境默认允许的源
		allowedOrigins = "http://localhost:8082,http://127.0.0.1:8082"
	}
	
	// 检查Origin是否在允许列表中
	for _, allowed := range strings.Split(allowedOrigins, ",") {
		allowed = strings.TrimSpace(allowed)
		if allowed == origin {
			return true
		}
		
		// 支持通配符域名（如 *.example.com）
		if strings.HasPrefix(allowed, "*.") {
			domain := allowed[2:]
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}
	return false
}
