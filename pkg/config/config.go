package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// ToolkitConfig 是根配置结构
type ToolkitConfig struct {
	PluginDirs []string                `yaml:"plugin_dirs"`
	Plugins    map[string]PluginConfig `yaml:"plugins"`
	Logging    LogConfig               `yaml:"logging"`
	General    GeneralConfig           `yaml:"general"`
}

// PluginConfig 保存每个插件的配置
type PluginConfig struct {
	Enabled    bool                   `yaml:"enabled"`
	Path       string                 `yaml:"path,omitempty"`
	RemoteAddr string                 `yaml:"remote_addr,omitempty"`
	Config     map[string]interface{} `yaml:"config,omitempty"`
}

// LogConfig 保存日志配置
type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	File   string `yaml:"file,omitempty"`
}

// GeneralConfig 保存常规设置
type GeneralConfig struct {
	AutoDiscover bool `yaml:"auto_discover"`
	HotReload    bool `yaml:"hot_reload"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *ToolkitConfig {
	return &ToolkitConfig{
		PluginDirs: []string{"./plugins"},
		Plugins:    make(map[string]PluginConfig),
		Logging: LogConfig{
			Level:  "info",
			Format: "text",
		},
		General: GeneralConfig{
			AutoDiscover: true,
			HotReload:    false,
		},
	}
}

// Manager 配置管理器接口
type Manager interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}) error
	Save() error
	Reload() error
	GetPluginConfig(pluginName string) (map[string]interface{}, bool)
	SetPluginConfig(pluginName string, config map[string]interface{}) error
	GetConfig() *ToolkitConfig
}

// YamlConfigManager YAML配置管理器实现
type YamlConfigManager struct {
	config     *ToolkitConfig
	configPath string
	mu         sync.RWMutex
}

// NewYamlConfigManager 创建YAML配置管理器
func NewYamlConfigManager(configPath string) (*YamlConfigManager, error) {
	manager := &YamlConfigManager{
		configPath: configPath,
	}

	if err := manager.Reload(); err != nil {
		return nil, err
	}

	return manager, nil
}

// Get 获取配置值
func (m *YamlConfigManager) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	switch key {
	case "plugin_dirs":
		return m.config.PluginDirs, true
	case "logging":
		return m.config.Logging, true
	case "general":
		return m.config.General, true
	default:
		return nil, false
	}
}

// Set 设置配置值
func (m *YamlConfigManager) Set(key string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch key {
	case "plugin_dirs":
		if dirs, ok := value.([]string); ok {
			m.config.PluginDirs = dirs
			return nil
		}
		return fmt.Errorf("invalid type for plugin_dirs: expected []string")
	case "logging.level":
		if level, ok := value.(string); ok {
			m.config.Logging.Level = level
			return nil
		}
		return fmt.Errorf("invalid type for logging.level: expected string")
	case "logging.format":
		if format, ok := value.(string); ok {
			m.config.Logging.Format = format
			return nil
		}
		return fmt.Errorf("invalid type for logging.format: expected string")
	case "logging.file":
		if file, ok := value.(string); ok {
			m.config.Logging.File = file
			return nil
		}
		return fmt.Errorf("invalid type for logging.file: expected string")
	case "general.auto_discover":
		if autoDiscover, ok := value.(bool); ok {
			m.config.General.AutoDiscover = autoDiscover
			return nil
		}
		return fmt.Errorf("invalid type for general.auto_discover: expected bool")
	case "general.hot_reload":
		if hotReload, ok := value.(bool); ok {
			m.config.General.HotReload = hotReload
			return nil
		}
		return fmt.Errorf("invalid type for general.hot_reload: expected bool")
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
}

// Save 保存配置到文件
func (m *YamlConfigManager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return SaveConfig(m.config, m.configPath)
}

// Reload 从文件重新加载配置
func (m *YamlConfigManager) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg, err := LoadConfig(m.configPath)
	if err != nil {
		return err
	}

	m.config = cfg
	return nil
}

// GetPluginConfig 获取插件配置
func (m *YamlConfigManager) GetPluginConfig(pluginName string) (map[string]interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if cfg, exists := m.config.Plugins[pluginName]; exists {
		return cfg.Config, true
	}

	return nil, false
}

// SetPluginConfig 设置插件配置
func (m *YamlConfigManager) SetPluginConfig(pluginName string, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.config.Plugins[pluginName]; !exists {
		m.config.Plugins[pluginName] = PluginConfig{
			Enabled: true,
			Config:  make(map[string]interface{}),
		}
	}

	pluginConfig := m.config.Plugins[pluginName]
	pluginConfig.Config = config
	m.config.Plugins[pluginName] = pluginConfig

	return nil
}

// GetConfig 获取完整配置
func (m *YamlConfigManager) GetConfig() *ToolkitConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.config
}

// LoadConfig 从YAML文件加载配置
func LoadConfig(path string) (*ToolkitConfig, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// SaveConfig 保存配置到YAML文件
func SaveConfig(cfg *ToolkitConfig, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
