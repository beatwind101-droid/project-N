package core

import (
	tkplugin "github.com/yourorg/toolkit/pkg/plugin"
)

// PluginLoader 负责插件加载
type PluginLoader interface {
	LoadPlugins() error
	GetPlugin(name string) (*ManagedPlugin, error)
	ListPlugins() map[string]*ManagedPlugin
}

// PluginExecutor 负责插件执行
type PluginExecutor interface {
	ExecutePlugin(name string, params map[string]interface{}) (*tkplugin.Result, error)
}

// PluginLifecycle 负责插件生命周期管理
type PluginLifecycle interface {
	InitializePlugin(name string, config map[string]interface{}) error
	InitializeAll(configs map[string]map[string]interface{}) error
	ShutdownPlugin(name string) error
	ShutdownAll()
}

// PluginRegistry 负责插件注册和查询
type PluginRegistry interface {
	RegisterPlugin(mp *ManagedPlugin) error
	UnregisterPlugin(name string) error
	GetPlugin(name string) (*ManagedPlugin, error)
	ListPlugins() map[string]*ManagedPlugin
}

// PluginManager 整合所有接口
type PluginManager interface {
	PluginLoader
	PluginExecutor
	PluginLifecycle
	PluginRegistry
}
