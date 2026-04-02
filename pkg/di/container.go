package di

import (
	"fmt"
	"sync"
)

// Container 依赖注入容器
type Container struct {
	services map[string]interface{}
	mu       sync.RWMutex
}

// NewContainer 创建新的依赖注入容器
func NewContainer() *Container {
	return &Container{
		services: make(map[string]interface{}),
	}
}

// Register 注册服务
func (c *Container) Register(name string, service interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.services[name] = service
}

// Get 获取服务
func (c *Container) Get(name string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	service, exists := c.services[name]
	return service, exists
}

// MustGet 获取服务，如果不存在则panic
func (c *Container) MustGet(name string) interface{} {
	service, exists := c.Get(name)
	if !exists {
		panic(fmt.Sprintf("service %s not found in container", name))
	}
	return service
}

// GetConfigManager 获取配置管理器
func (c *Container) GetConfigManager() (interface{}, bool) {
	return c.Get("config_manager")
}

// GetPluginManager 获取插件管理器
func (c *Container) GetPluginManager() (interface{}, bool) {
	return c.Get("plugin_manager")
}

// GetLogger 获取日志记录器
func (c *Container) GetLogger() (interface{}, bool) {
	return c.Get("logger")
}

// MustGetConfigManager 获取配置管理器，如果不存在则panic
func (c *Container) MustGetConfigManager() interface{} {
	return c.MustGet("config_manager")
}

// MustGetPluginManager 获取插件管理器，如果不存在则panic
func (c *Container) MustGetPluginManager() interface{} {
	return c.MustGet("plugin_manager")
}

// MustGetLogger 获取日志记录器，如果不存在则panic
func (c *Container) MustGetLogger() interface{} {
	return c.MustGet("logger")
}

// ListServices 列出所有已注册的服务
func (c *Container) ListServices() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	services := make([]string, 0, len(c.services))
	for name := range c.services {
		services = append(services, name)
	}
	return services
}
