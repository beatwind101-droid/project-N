package core

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hashicorp/go-hclog"

	tkplugin "github.com/yourorg/toolkit/pkg/plugin"
)

const PluginTypeGo = "go"
const PluginTypeExe = "exe"

// 插件文件大小限制（100MB）
const MaxPluginSize = 100 * 1024 * 1024

// PluginDiscovery handles plugin discovery.
type PluginDiscovery struct {
	pluginDirs []string
	logger     hclog.Logger
}

// NewPluginDiscovery creates a new plugin discovery.
func NewPluginDiscovery(pluginDirs []string, logger hclog.Logger) *PluginDiscovery {
	if logger == nil {
		logger = hclog.Default()
	}
	return &PluginDiscovery{
		pluginDirs: pluginDirs,
		logger:     logger.Named("plugin-discovery"),
	}
}

// Discover discovers all plugins in the configured directories.
func (d *PluginDiscovery) Discover() ([]tkplugin.PluginInfo, error) {
	var infos []tkplugin.PluginInfo

	for _, dir := range d.pluginDirs {
		if err := d.discoverRecursive(dir, &infos); err != nil {
			d.logger.Warn("failed to read plugin directory", "dir", dir, "error", err)
		}
	}

	return infos, nil
}

// discoverRecursive recursively discovers plugins in directories
func (d *PluginDiscovery) discoverRecursive(dir string, infos *[]tkplugin.PluginInfo) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			// Recursively scan subdirectories
			if err := d.discoverRecursive(path, infos); err != nil {
				d.logger.Warn("failed to scan subdirectory", "dir", path, "error", err)
			}
		} else {
			// Process files
			info, err := d.discoverPlugin(path)
			if err != nil {
				d.logger.Warn("failed to discover plugin", "path", path, "error", err)
				continue
			}

			if info != nil {
				*infos = append(*infos, *info)
			}
		}
	}

	return nil
}

// validatePluginFile 验证插件文件的大小和类型
func validatePluginFile(path string) error {
	// 检查文件大小
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	
	if info.Size() > MaxPluginSize {
		return fmt.Errorf("plugin file too large: %d bytes (max: %d bytes)", info.Size(), MaxPluginSize)
	}
	
	// 检查文件是否为普通文件
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", path)
	}
	
	// 检查文件权限（只允许读取和执行权限）
	// 在Windows系统中，不检查执行权限
	if runtime.GOOS != "windows" {
		mode := info.Mode()
		if mode&0111 == 0 { // 无执行权限
			ext := filepath.Ext(path)
			if ext == ".exe" || ext == "" {
				return fmt.Errorf("executable plugin requires execute permission")
			}
		}
	}
	
	return nil
}

func (d *PluginDiscovery) discoverPlugin(path string) (*tkplugin.PluginInfo, error) {
	// 验证插件文件
	if err := validatePluginFile(path); err != nil {
		return nil, err
	}
	
	name := filepath.Base(path)
	ext := filepath.Ext(name)

	if ext == ".so" {
		return &tkplugin.PluginInfo{
			Name:       strings.TrimSuffix(name, ext),
			Path:       path,
			PluginType: PluginTypeGo,
			Enabled:    true,
			Metadata: tkplugin.ToolMetadata{
				Version:     "1.0.0",
				Description: "Go native plugin",
				Author:      "unknown",
				Category:    "general",
				Tags:        []string{"plugin"},
			},
		}, nil
	}

	if ext == ".exe" || ext == "" {
		return &tkplugin.PluginInfo{
			Name:       strings.TrimSuffix(name, ext),
			Path:       path,
			PluginType: PluginTypeExe,
			Enabled:    true,
			Metadata: tkplugin.ToolMetadata{
				Version:     "1.0.0",
				Description: "Executable plugin",
				Author:      "unknown",
				Category:    "general",
				Tags:        []string{"plugin"},
			},
		}, nil
	}

	return nil, fmt.Errorf("unsupported plugin type: %s", path)
}
