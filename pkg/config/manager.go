package config

import (
	"sync"
)

// ConfigManager 配置管理器接口
type ConfigManager interface {
	// Get 获取配置值
	Get(key string) (interface{}, bool)
	// Set 设置配置值
	Set(key string, value interface{}) error
	// Save 保存配置
	Save() error
	// Reload 重新加载配置
	Reload() error
	// GetPluginConfig 获取插件配置
	GetPluginConfig(pluginName string) (map[string]interface{}, bool)
	// SetPluginConfig 设置插件配置
	SetPluginConfig(pluginName string, config map[string]interface{}) error
}

// SimpleConfigManager 简单的配置管理器实现
type SimpleConfigManager struct {
	config map[string]interface{}
	mu     sync.RWMutex
}

// NewSimpleConfigManager 创建简单配置管理器
func NewSimpleConfigManager() *SimpleConfigManager {
	return &SimpleConfigManager{
		config: make(map[string]interface{}),
	}
}

// Get 获取配置值
func (m *SimpleConfigManager) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.config[key]
	return val, ok
}

// Set 设置配置值
func (m *SimpleConfigManager) Set(key string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config[key] = value
	return nil
}

// Save 保存配置（空实现）
func (m *SimpleConfigManager) Save() error {
	return nil
}

// Reload 重新加载配置（空实现）
func (m *SimpleConfigManager) Reload() error {
	return nil
}

// GetPluginConfig 获取插件配置
func (m *SimpleConfigManager) GetPluginConfig(pluginName string) (map[string]interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if val, ok := m.config["plugin."+pluginName]; ok {
		if cfg, ok := val.(map[string]interface{}); ok {
			return cfg, true
		}
	}

	return nil, false
}

// SetPluginConfig 设置插件配置
func (m *SimpleConfigManager) SetPluginConfig(pluginName string, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config["plugin."+pluginName] = config
	return nil
}
