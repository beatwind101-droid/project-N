package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/yourorg/toolkit/pkg/common"

	"github.com/hashicorp/go-hclog"
)

// Handler provides HTTP handlers for MCP protocol
type Handler struct {
	server   *Server
	logger   hclog.Logger
	sessions sync.Map
}

// NewHandler creates a new MCP HTTP handler
func NewHandler(server *Server, logger hclog.Logger) *Handler {
	if logger == nil {
		logger = hclog.Default()
	}
	return &Handler{
		server: server,
		logger: logger.Named("mcp"),
	}
}

// RegisterRoutes registers MCP routes to an HTTP mux
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/mcp", h.handleMCP)
	mux.HandleFunc("/mcp/sse", h.handleSSE)
	mux.HandleFunc("/mcp/health", h.handleHealth)
}

// handleMCP handles JSON-RPC over HTTP POST
func (h *Handler) handleMCP(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	origin := r.Header.Get("Origin")
	if origin != "" && common.IsValidOrigin(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Mcp-Session-Id")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, nil, ParseError, "Parse error")
		return
	}

	h.logger.Debug("MCP request", "method", req.Method, "id", req.ID)

	resp := h.server.HandleRequest(req)
	h.logger.Debug("MCP response", "method", req.Method, "hasError", resp.Error != nil)

	// For notifications, return 204 No Content
	if req.Method == "notifications/initialized" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

// handleSSE handles Server-Sent Events for streaming
func (h *Handler) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	origin := r.Header.Get("Origin")
	if origin != "" && common.IsValidOrigin(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}

	// Send server info
	serverInfo := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "server/info",
		"params": map[string]interface{}{
			"name":    h.server.name,
			"version": h.server.version,
		},
	}
	h.writeSSE(w, flusher, "server_info", serverInfo)

	// Keep connection alive
	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		default:
			// Send heartbeat every 30 seconds would go here
			// For now, just wait
		}
	}
}

// handleHealth returns server health status
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"server":  h.server.name,
		"version": h.server.version,
		"tools":   len(h.server.tools),
	})
}

func (h *Handler) writeError(w http.ResponseWriter, id interface{}, code int, message string) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &Error{Code: code, Message: message},
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) writeSSE(w http.ResponseWriter, flusher http.Flusher, event string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(jsonData))
	flusher.Flush()
}

// SSEClient represents a connected SSE client
type SSEClient struct {
	id   string
	ch   chan []byte
	done chan struct{}
}

// SSEManager manages SSE connections
type SSEManager struct {
	mu      sync.RWMutex
	clients map[string]*SSEClient
}

// NewSSEManager creates a new SSE manager
func NewSSEManager() *SSEManager {
	return &SSEManager{
		clients: make(map[string]*SSEClient),
	}
}

// Add adds a new SSE client
func (m *SSEManager) Add(id string, client *SSEClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[id] = client
}

// Remove removes an SSE client
func (m *SSEManager) Remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, id)
}

// Broadcast sends data to all SSE clients
func (m *SSEManager) Broadcast(data []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, client := range m.clients {
		select {
		case client.ch <- data:
		default:
			// Client buffer full, skip
		}
	}
}

// handleSSEWithBody handles SSE with request body (for JSON-RPC over SSE)
func (h *Handler) handleSSEWithBody(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	origin := r.Header.Get("Origin")
	if origin != "" && common.IsValidOrigin(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}

	scanner := bufio.NewScanner(r.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			h.writeSSE(w, flusher, "error", map[string]string{"error": "Invalid JSON"})
			continue
		}

		resp := h.server.HandleRequest(req)
		jsonResp, _ := json.Marshal(resp)
		h.writeSSE(w, flusher, "response", json.RawMessage(jsonResp))
	}
}

// GetTools returns the list of available tools (for external use)
func (h *Handler) GetTools() []Tool {
	return h.server.handleToolsList().Tools
}

// CallTool calls a tool directly (for external use)
func (h *Handler) CallTool(name string, args map[string]interface{}) (*ToolCallResult, error) {
	params := ToolCallParams{Name: name, Arguments: args}
	result, rpcErr := h.server.handleToolsCall(params)
	if rpcErr != nil {
		return nil, fmt.Errorf("%s", rpcErr.Message)
	}
	return result, nil
}

// MCPInfo returns MCP server info for discovery
func (h *Handler) MCPInfo() map[string]interface{} {
	tools := h.GetTools()
	toolNames := make([]string, len(tools))
	for i, t := range tools {
		toolNames[i] = t.Name
	}

	return map[string]interface{}{
		"protocol":  "mcp",
		"version":   "2024-11-05",
		"server":    h.server.name,
		"transport": "http",
		"endpoints": map[string]string{
			"rpc":    "/mcp",
			"sse":    "/mcp/sse",
			"health": "/mcp/health",
		},
		"tools": toolNames,
	}
}

// IsMCPRequest checks if the request is an MCP request
func IsMCPRequest(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	contentType := r.Header.Get("Content-Type")
	return strings.Contains(accept, "text/event-stream") ||
		strings.Contains(contentType, "application/json")
}
