package plugin

import "context"

// Tool is the core interface that all plugins must implement.
type Tool interface {
	Metadata() ToolMetadata
	Init(ctx context.Context, config map[string]interface{}) error
	Execute(ctx context.Context, params map[string]interface{}) (*Result, error)
	Validate(params map[string]interface{}) error
	Shutdown(ctx context.Context) error
}

// ToolMetadata contains plugin metadata.
type ToolMetadata struct {
	Name         string           `json:"name" yaml:"name"`
	Version      string           `json:"version" yaml:"version"`
	Description  string           `json:"description" yaml:"description"`
	Author       string           `json:"author" yaml:"author"`
	Tags         []string         `json:"tags" yaml:"tags"`
	Category     string           `json:"category" yaml:"category"`
	ConfigSchema map[string]Field `json:"config_schema,omitempty" yaml:"config_schema,omitempty"`
}

// Field defines a configuration field schema.
type Field struct {
	Type        string      `json:"type" yaml:"type"`
	Description string      `json:"description" yaml:"description"`
	Required    bool        `json:"required" yaml:"required"`
	Default     interface{} `json:"default,omitempty" yaml:"default,omitempty"`
}

// Result represents the execution result of a tool.
type Result struct {
	Success bool                   `json:"success"`
	Data    interface{}            `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Metrics map[string]interface{} `json:"metrics,omitempty"`
}

// PluginInfo holds discovered plugin information.
type PluginInfo struct {
	Name       string       `json:"name"`
	Path       string       `json:"path"`
	PluginType string       `json:"plugin_type,omitempty"`
	Metadata   ToolMetadata `json:"metadata"`
	Enabled    bool         `json:"enabled"`
}

// PluginState represents the lifecycle state of a plugin.
type PluginState int

const (
	StateUnknown PluginState = iota
	StateLoaded
	StateInitialized
	StateRunning
	StateStopped
	StateError
)

func (s PluginState) String() string {
	switch s {
	case StateLoaded:
		return "loaded"
	case StateInitialized:
		return "initialized"
	case StateRunning:
		return "running"
	case StateStopped:
		return "stopped"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}
