package core

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"plugin"
	"strings"
	"sync"

	"github.com/hashicorp/go-hclog"
	hashicorpPlugin "github.com/hashicorp/go-plugin"

	tkplugin "github.com/yourorg/toolkit/pkg/plugin"
)

// PluginManagerImpl 插件管理器实现
type PluginManagerImpl struct {
	mu        sync.RWMutex
	plugins   map[string]*ManagedPlugin
	logger    hclog.Logger
	discovery *PluginDiscovery
	config    *ManagerConfig
}

// ManagerConfig 管理器配置
type ManagerConfig struct {
	PluginDirs []string `yaml:"plugin_dirs"`
	ConfigFile string   `yaml:"config_file"`
}

// ManagedPlugin 托管的插件
type ManagedPlugin struct {
	Info   tkplugin.PluginInfo
	State  tkplugin.PluginState
	Client *hashicorpPlugin.Client
	Tool   tkplugin.Tool
	Config map[string]interface{}
}

// NewPluginManager 创建新的插件管理器
func NewPluginManager(cfg *ManagerConfig, logger hclog.Logger) *PluginManagerImpl {
	if logger == nil {
		logger = hclog.Default()
	}
	return &PluginManagerImpl{
		plugins:   make(map[string]*ManagedPlugin),
		logger:    logger.Named("plugin-manager"),
		discovery: NewPluginDiscovery(cfg.PluginDirs, logger),
		config:    cfg,
	}
}

// LoadPlugins 发现并加载所有配置目录中的插件
func (m *PluginManagerImpl) LoadPlugins() error {
	pluginInfos, err := m.discovery.Discover()
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	for _, info := range pluginInfos {
		if err := m.loadPlugin(info); err != nil {
			m.logger.Warn("failed to load plugin", "name", info.Name, "error", err)
			continue
		}
	}

	m.logger.Info("plugins loaded", "count", len(m.plugins))
	return nil
}

func (m *PluginManagerImpl) loadPlugin(info tkplugin.PluginInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.plugins[info.Name]; exists {
		return fmt.Errorf("plugin %s already loaded", info.Name)
	}

	pluginType := info.PluginType
	if pluginType == "" {
		pluginType = getPluginTypeFromPath(info.Path)
	}

	var tool tkplugin.Tool
	var client *hashicorpPlugin.Client

	if pluginType == PluginTypeGo {
		goTool, err := m.loadGoPlugin(info)
		if err != nil {
			return fmt.Errorf("failed to load go plugin: %w", err)
		}
		tool = goTool
	} else {
		var err error
		tool, client, err = m.loadExePlugin(info)
		if err != nil {
			return fmt.Errorf("failed to load exe plugin: %w", err)
		}
	}

	m.plugins[info.Name] = &ManagedPlugin{
		Info:   info,
		State:  tkplugin.StateLoaded,
		Client: client,
		Tool:   tool,
	}

	m.logger.Info("plugin loaded", "name", info.Name, "version", tool.Metadata().Version)
	return nil
}

func getPluginTypeFromPath(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".go", ".so", ".dll", ".dylib":
		return PluginTypeGo
	default:
		return PluginTypeExe
	}
}

func (m *PluginManagerImpl) loadGoPlugin(info tkplugin.PluginInfo) (tkplugin.Tool, error) {
	if err := m.validatePluginPath(info.Path); err != nil {
		return nil, fmt.Errorf("plugin path validation failed: %w", err)
	}

	securePath, err := securePluginPath(info.Path)
	if err != nil {
		return nil, fmt.Errorf("plugin path cleanup failed: %w", err)
	}

	p, err := plugin.Open(securePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %w", err)
	}

	sym, err := p.Lookup("Tool")
	if err != nil {
		return nil, fmt.Errorf("symbol Tool not found: %w", err)
	}

	tool, ok := sym.(tkplugin.Tool)
	if !ok {
		return nil, fmt.Errorf("symbol Tool does not implement tkplugin.Tool interface")
	}

	return tool, nil
}

func (m *PluginManagerImpl) loadExePlugin(info tkplugin.PluginInfo) (tkplugin.Tool, *hashicorpPlugin.Client, error) {
	if err := m.validatePluginPath(info.Path); err != nil {
		return nil, nil, fmt.Errorf("plugin path validation failed: %w", err)
	}

	securePath, err := securePluginPath(info.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("plugin path cleanup failed: %w", err)
	}

	client := hashicorpPlugin.NewClient(&hashicorpPlugin.ClientConfig{
		HandshakeConfig: hashicorpPlugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   tkplugin.PluginMagicCookieKey,
			MagicCookieValue: tkplugin.PluginMagicCookieValue,
		},
		Plugins: map[string]hashicorpPlugin.Plugin{
			"tool": &tkplugin.ToolkitRPCPlugin{},
		},
		Cmd:    exec.Command(securePath),
		Logger: m.logger.Named(info.Name),
	})

	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, nil, fmt.Errorf("failed to create rpc client: %w", err)
	}

	raw, err := rpcClient.Dispense("tool")
	if err != nil {
		client.Kill()
		return nil, nil, fmt.Errorf("failed to dispense tool: %w", err)
	}

	tool, ok := raw.(tkplugin.Tool)
	if !ok {
		client.Kill()
		return nil, nil, fmt.Errorf("dispensed object does not implement Tool interface")
	}

	return tool, client, nil
}

// InitializePlugin 使用配置初始化已加载的插件
func (m *PluginManagerImpl) InitializePlugin(name string, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mp, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	ctx := context.Background()
	if err := mp.Tool.Init(ctx, config); err != nil {
		mp.State = tkplugin.StateError
		return fmt.Errorf("init failed for %s: %w", name, err)
	}

	mp.State = tkplugin.StateInitialized
	mp.Config = config
	return nil
}

// InitializeAll 初始化所有插件
func (m *PluginManagerImpl) InitializeAll(configs map[string]map[string]interface{}) error {
	var errs []error

	for name, config := range configs {
		if err := m.InitializePlugin(name, config); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("some plugins failed to initialize: %v", errs)
	}
	return nil
}

// ExecutePlugin 执行插件
func (m *PluginManagerImpl) ExecutePlugin(name string, params map[string]interface{}) (*tkplugin.Result, error) {
	m.mu.Lock()
	mp, exists := m.plugins[name]
	if !exists {
		m.mu.Unlock()
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	if mp.State < tkplugin.StateInitialized {
		m.mu.Unlock()
		return nil, fmt.Errorf("plugin %s not initialized (state: %s)", name, mp.State)
	}

	mp.State = tkplugin.StateRunning
	m.mu.Unlock()

	ctx := context.Background()
	result, err := mp.Tool.Execute(ctx, params)

	m.mu.Lock()
	if err != nil {
		mp.State = tkplugin.StateError
	} else {
		mp.State = tkplugin.StateInitialized
	}
	m.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("execution failed for %s: %w", name, err)
	}
	return result, nil
}

// GetPlugin 获取插件
func (m *PluginManagerImpl) GetPlugin(name string) (*ManagedPlugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mp, exists := m.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}
	return mp, nil
}

// ListPlugins 列出所有插件
func (m *PluginManagerImpl) ListPlugins() map[string]*ManagedPlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*ManagedPlugin, len(m.plugins))
	for k, v := range m.plugins {
		result[k] = v
	}
	return result
}

// RegisterPlugin 注册插件
func (m *PluginManagerImpl) RegisterPlugin(mp *ManagedPlugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.plugins[mp.Info.Name]; exists {
		return fmt.Errorf("plugin %s already registered", mp.Info.Name)
	}

	m.plugins[mp.Info.Name] = mp
	return nil
}

// UnregisterPlugin 注销插件
func (m *PluginManagerImpl) UnregisterPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.plugins[name]; !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	delete(m.plugins, name)
	return nil
}

// ShutdownPlugin 关闭插件
func (m *PluginManagerImpl) ShutdownPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mp, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	if mp.Client != nil {
		mp.Client.Kill()
	}

	mp.State = tkplugin.StateStopped
	return nil
}

// ShutdownAll 关闭所有插件
func (m *PluginManagerImpl) ShutdownAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, mp := range m.plugins {
		if mp.Client != nil {
			mp.Client.Kill()
		}
		mp.State = tkplugin.StateStopped
		m.logger.Info("plugin shutdown", "name", name)
	}
}

func (m *PluginManagerImpl) validatePluginPath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	
	// 检查是否在配置的插件目录内
	allowed := false
	for _, dir := range m.config.PluginDirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		// 确保路径在允许目录内
		if strings.HasPrefix(absPath, absDir) {
			allowed = true
			break
		}
	}
	
	if !allowed {
		return fmt.Errorf("plugin path %q is outside allowed directories", path)
	}
	
	// 额外检查：禁止系统目录
	systemDirs := []string{
		`C:\Windows`,
		`C:\Program Files`,
		`/etc`,
		`/usr/bin`,
	}
	for _, sysDir := range systemDirs {
		if strings.HasPrefix(absPath, sysDir) {
			return fmt.Errorf("loading plugins from system directories is forbidden")
		}
	}
	
	return nil
}

func securePluginPath(path string) (string, error) {
	cleanPath := filepath.Clean(path)
	if !filepath.IsAbs(cleanPath) {
		cleanPath = filepath.Join(".", cleanPath)
	}
	return cleanPath, nil
}
