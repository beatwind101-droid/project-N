package core

import (
	"errors"
	"fmt"
)

// 插件相关错误
var (
	ErrPluginNotFound        = errors.New("plugin not found")
	ErrPluginAlreadyLoaded   = errors.New("plugin already loaded")
	ErrPluginNotLoaded       = errors.New("plugin not loaded")
	ErrPluginNotInitialized  = errors.New("plugin not initialized")
	ErrPluginExecutionFailed = errors.New("plugin execution failed")
	ErrPluginInitFailed      = errors.New("plugin initialization failed")
	ErrPluginShutdownFailed  = errors.New("plugin shutdown failed")
	ErrInvalidPluginType     = errors.New("invalid plugin type")
	ErrInvalidPluginPath     = errors.New("invalid plugin path")
	ErrInvalidPluginName     = errors.New("invalid plugin name")
)

// PluginError 插件错误结构体
type PluginError struct {
	PluginName string
	Op         string
	Err        error
}

func (e *PluginError) Error() string {
	if e.PluginName == "" {
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("%s: plugin %q: %v", e.Op, e.PluginName, e.Err)
}

func (e *PluginError) Unwrap() error {
	return e.Err
}

// NewPluginError 创建新的插件错误
func NewPluginError(pluginName, op string, err error) error {
	return &PluginError{
		PluginName: pluginName,
		Op:         op,
		Err:        err,
	}
}

// IsPluginNotFound 检查是否为插件未找到错误
func IsPluginNotFound(err error) bool {
	return errors.Is(err, ErrPluginNotFound)
}

// IsPluginAlreadyLoaded 检查是否为插件已加载错误
func IsPluginAlreadyLoaded(err error) bool {
	return errors.Is(err, ErrPluginAlreadyLoaded)
}

// IsPluginExecutionFailed 检查是否为插件执行失败错误
func IsPluginExecutionFailed(err error) bool {
	return errors.Is(err, ErrPluginExecutionFailed)
}
