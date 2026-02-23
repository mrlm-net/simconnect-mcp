package mcpadapter

import (
	"context"
	"encoding/json"
	"fmt"
)

// ── JSON-RPC 2.0 types ───────────────────────────────────────────────────────

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"` // absent or null for notifications
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// isNotification returns true when the request has no ID (client does not expect a reply).
func (r *rpcRequest) isNotification() bool {
	return len(r.ID) == 0 || string(r.ID) == "null"
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func okResp(id json.RawMessage, result any) *rpcResponse {
	return &rpcResponse{JSONRPC: "2.0", ID: id, Result: result}
}

func errResp(id json.RawMessage, code int, msg string) *rpcResponse {
	return &rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}}
}

// ── MCP dispatch ─────────────────────────────────────────────────────────────

func (s *Server) dispatch(ctx context.Context, req *rpcRequest) *rpcResponse {
	switch req.Method {
	case "initialize":
		return s.mcpInitialize(req)
	case "tools/list":
		return s.mcpListTools(req)
	case "tools/call":
		return s.mcpCallTool(ctx, req)
	case "ping":
		return okResp(req.ID, struct{}{})
	default:
		return errResp(req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
	}
}

func (s *Server) mcpInitialize(req *rpcRequest) *rpcResponse {
	type result struct {
		ProtocolVersion string `json:"protocolVersion"`
		ServerInfo      struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
		Capabilities struct {
			Tools struct{} `json:"tools"`
		} `json:"capabilities"`
	}
	r := result{}
	r.ProtocolVersion = "2024-11-05"
	r.ServerInfo.Name = s.name
	r.ServerInfo.Version = s.version
	return okResp(req.ID, r)
}

func (s *Server) mcpListTools(req *rpcRequest) *rpcResponse {
	s.mu.RLock()
	tools := make([]Tool, len(s.tools))
	copy(tools, s.tools)
	s.mu.RUnlock()
	return okResp(req.ID, map[string]any{"tools": tools})
}

// callToolParams mirrors the MCP tools/call params structure.
type callToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

func (s *Server) mcpCallTool(ctx context.Context, req *rpcRequest) *rpcResponse {
	var params callToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errResp(req.ID, -32602, "invalid params")
	}

	s.mu.RLock()
	handler, ok := s.handlers[params.Name]
	s.mu.RUnlock()

	if !ok {
		return errResp(req.ID, -32602, fmt.Sprintf("unknown tool: %s", params.Name))
	}

	result, err := handler(ctx, params.Arguments)
	if err != nil {
		return okResp(req.ID, &CallToolResult{
			Content: []Content{{Type: "text", Text: err.Error()}},
			IsError: true,
		})
	}
	if result == nil {
		result = &CallToolResult{Content: []Content{{Type: "text", Text: ""}}}
	}
	return okResp(req.ID, result)
}
