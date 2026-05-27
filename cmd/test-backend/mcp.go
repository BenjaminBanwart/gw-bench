package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"crypto/rand"
)

// MCP JSON-RPC types
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	ID      json.RawMessage `json:"id"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpInitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ServerInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

type mcpTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type mcpToolsListResult struct {
	Tools []mcpTool `json:"tools"`
}

type mcpToolCallResult struct {
	Content []mcpContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

type mcpContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body to 1MB to prevent OOM from oversized payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req jsonRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONRPCError(w, nil, -32700, "parse error")
		return
	}

	if req.JSONRPC != "2.0" {
		writeJSONRPCError(w, req.ID, -32600, "invalid request: jsonrpc must be 2.0")
		return
	}

	switch req.Method {
	case "initialize":
		handleInitialize(w, req)
	case "tools/list":
		handleToolsList(w, req)
	case "tools/call":
		handleToolsCall(w, req)
	default:
		writeJSONRPCError(w, req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
	}
}

func handleInitialize(w http.ResponseWriter, req jsonRPCRequest) {
	// Generate session ID
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	sessionID := fmt.Sprintf("%x", b)

	result := mcpInitializeResult{
		ProtocolVersion: "2025-03-26",
		Capabilities: map[string]any{
			"tools": map[string]any{},
		},
	}
	result.ServerInfo.Name = "gw-bench-test-backend"
	result.ServerInfo.Version = "0.1.0"

	w.Header().Set("Mcp-Session", sessionID)
	writeJSONRPCResult(w, req.ID, result)
}

func handleToolsList(w http.ResponseWriter, req jsonRPCRequest) {
	result := mcpToolsListResult{
		Tools: []mcpTool{
			{
				Name:        "echo",
				Description: "Echoes back the input message",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"message": map[string]any{
							"type":        "string",
							"description": "The message to echo back",
						},
					},
					"required": []string{"message"},
				},
			},
			{
				Name:        "delay",
				Description: "Waits for the specified duration then returns",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"ms": map[string]any{
							"type":        "integer",
							"description": "Milliseconds to delay",
						},
					},
					"required": []string{"ms"},
				},
			},
		},
	}
	writeJSONRPCResult(w, req.ID, result)
}

func handleToolsCall(w http.ResponseWriter, req jsonRPCRequest) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeJSONRPCError(w, req.ID, -32602, "invalid params")
		return
	}

	switch params.Name {
	case "echo":
		var args struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			writeJSONRPCError(w, req.ID, -32602, "invalid arguments for echo")
			return
		}
		writeJSONRPCResult(w, req.ID, mcpToolCallResult{
			Content: []mcpContent{{Type: "text", Text: args.Message}},
		})

	case "delay":
		var args struct {
			Ms json.RawMessage `json:"ms"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			writeJSONRPCError(w, req.ID, -32602, "invalid arguments for delay")
			return
		}
		ms, err := strconv.Atoi(string(args.Ms))
		if err != nil {
			writeJSONRPCError(w, req.ID, -32602, "ms must be an integer")
			return
		}
		if ms < 0 || ms > 30000 {
			writeJSONRPCError(w, req.ID, -32602, "ms must be between 0 and 30000")
			return
		}
		time.Sleep(time.Duration(ms) * time.Millisecond)
		writeJSONRPCResult(w, req.ID, mcpToolCallResult{
			Content: []mcpContent{{Type: "text", Text: fmt.Sprintf("delayed %dms", ms)}},
		})

	default:
		writeJSONRPCError(w, req.ID, -32602, fmt.Sprintf("unknown tool: %s", params.Name))
	}
}

func writeJSONRPCResult(w http.ResponseWriter, id json.RawMessage, result any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func writeJSONRPCError(w http.ResponseWriter, id json.RawMessage, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &jsonRPCError{Code: code, Message: message},
	})
}
