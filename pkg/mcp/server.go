package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	tkplugin "github.com/yourorg/toolkit/pkg/plugin"
)

// ToolAdapter adapts a plugin.Tool to MCP Tool interface
type ToolAdapter interface {
	GetName() string
	GetDescription() string
	GetInputSchema() InputSchema
	Call(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error)
}

// Server represents an MCP server
type Server struct {
	name    string
	version string
	tools   map[string]ToolAdapter
	mu      sync.RWMutex
}

// NewServer creates a new MCP server
func NewServer(name, version string) *Server {
	return &Server{
		name:    name,
		version: version,
		tools:   make(map[string]ToolAdapter),
	}
}

// RegisterTool registers a tool adapter
func (s *Server) RegisterTool(adapter ToolAdapter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[adapter.GetName()] = adapter
}

// RegisterPlugin registers a plugin as an MCP tool
func (s *Server) RegisterPlugin(name string, tool tkplugin.Tool) {
	adapter := NewPluginToolAdapter(name, tool)
	s.RegisterTool(adapter)
}

// HandleRequest processes an MCP request and returns a response
func (s *Server) HandleRequest(req Request) Response {
	var result interface{}
	var rpcErr *Error

	switch req.Method {
	case "initialize":
		result = s.handleInitialize()
	case "notifications/initialized":
		// Notification, no response needed
		return Response{}
	case "tools/list":
		result = s.handleToolsList()
	case "tools/call":
		params, err := parseToolCallParams(req.Params)
		if err != nil {
			rpcErr = &Error{Code: InvalidParams, Message: err.Error()}
		} else {
			result, rpcErr = s.handleToolsCall(params)
		}
	case "resources/list":
		result = s.handleResourcesList()
	case "resources/read":
		result = s.handleResourcesRead(req.Params)
	case "prompts/list":
		result = s.handlePromptsList()
	case "prompts/get":
		result, rpcErr = s.handlePromptsGet(req.Params)
	default:
		rpcErr = &Error{Code: MethodNotFound, Message: fmt.Sprintf("Method not found: %s", req.Method)}
	}

	resp := Response{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	if rpcErr != nil {
		resp.Error = rpcErr
	} else {
		resp.Result = result
	}

	return resp
}

func (s *Server) handleInitialize() InitializeResult {
	return InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo: ServerInfo{
			Name:    s.name,
			Version: s.version,
		},
		Capabilities: Capabilities{
			Tools:     &ToolsCapability{ListChanged: true},
			Resources: &ResourcesCapability{Subscribe: false, ListChanged: false},
			Prompts:   &PromptsCapability{ListChanged: false},
		},
	}
}

func (s *Server) handleToolsList() ToolsListResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]Tool, 0, len(s.tools))
	for _, adapter := range s.tools {
		tools = append(tools, Tool{
			Name:        adapter.GetName(),
			Description: adapter.GetDescription(),
			InputSchema: adapter.GetInputSchema(),
		})
	}

	return ToolsListResult{Tools: tools}
}

func (s *Server) handleToolsCall(params ToolCallParams) (*ToolCallResult, *Error) {
	s.mu.RLock()
	adapter, exists := s.tools[params.Name]
	s.mu.RUnlock()

	if !exists {
		return nil, &Error{
			Code:    InvalidParams,
			Message: fmt.Sprintf("Tool not found: %s", params.Name),
		}
	}

	result, err := adapter.Call(context.Background(), params.Arguments)
	if err != nil {
		return &ToolCallResult{
			Content: []Content{{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
			IsError: true,
		}, nil
	}

	return result, nil
}

func (s *Server) handleResourcesList() ResourcesListResult {
	return ResourcesListResult{Resources: []Resource{}}
}

func (s *Server) handleResourcesRead(params interface{}) ResourceReadResult {
	return ResourceReadResult{Contents: []ResourceContent{}}
}

func (s *Server) handlePromptsList() PromptsListResult {
	return PromptsListResult{Prompts: []Prompt{}}
}

func (s *Server) handlePromptsGet(params interface{}) (*PromptGetResult, *Error) {
	return nil, &Error{Code: MethodNotFound, Message: "Prompts not implemented"}
}

// PluginToolAdapter adapts a plugin.Tool to MCP Tool interface
type PluginToolAdapter struct {
	name string
	tool tkplugin.Tool
}

// NewPluginToolAdapter creates a new adapter
func NewPluginToolAdapter(name string, tool tkplugin.Tool) *PluginToolAdapter {
	return &PluginToolAdapter{name: name, tool: tool}
}

func (a *PluginToolAdapter) GetName() string {
	return a.name
}

func (a *PluginToolAdapter) GetDescription() string {
	meta := a.tool.Metadata()
	return meta.Description
}

func (a *PluginToolAdapter) GetInputSchema() InputSchema {
	meta := a.tool.Metadata()
	schema := InputSchema{
		Type:       "object",
		Properties: make(map[string]Property),
	}

	for name, field := range meta.ConfigSchema {
		defaultValue := ""
		if field.Default != nil {
			defaultValue = fmt.Sprintf("%v", field.Default)
		}
		schema.Properties[name] = Property{
			Type:        field.Type,
			Description: field.Description,
			Default:     defaultValue,
		}
		if field.Required {
			schema.Required = append(schema.Required, name)
		}
	}

	return schema
}

func (a *PluginToolAdapter) Call(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
	result, err := a.tool.Execute(ctx, args)
	if err != nil {
		return &ToolCallResult{
			Content: []Content{{Type: "text", Text: fmt.Sprintf("Execution error: %v", err)}},
			IsError: true,
		}, nil
	}

	if !result.Success {
		return &ToolCallResult{
			Content: []Content{{Type: "text", Text: fmt.Sprintf("Failed: %s", result.Error)}},
			IsError: true,
		}, nil
	}

	// Format result data
	var text string
	switch data := result.Data.(type) {
	case string:
		text = data
	default:
		jsonBytes, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			text = fmt.Sprintf("%v", data)
		} else {
			text = string(jsonBytes)
		}
	}

	return &ToolCallResult{
		Content: []Content{{Type: "text", Text: text}},
	}, nil
}

func parseToolCallParams(params interface{}) (ToolCallParams, error) {
	data, err := json.Marshal(params)
	if err != nil {
		return ToolCallParams{}, err
	}

	var result ToolCallParams
	err = json.Unmarshal(data, &result)
	return result, err
}

// LogError logs an error
func LogError(format string, v ...interface{}) {
	log.Printf("[MCP] "+format, v...)
}
