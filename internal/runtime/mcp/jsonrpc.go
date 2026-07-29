package mcp

import "encoding/json"

const protocolVersion = "2024-11-05"

type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// * go_pkg_http.POSTStream takes map[string]any, so outbound frames stay maps
func newRequest(id int64, method string, params any) map[string]any {
	body := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}
	if params != nil {
		body["params"] = params
	}
	return body
}

func newNotification(method string, params any) map[string]any {
	body := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if params != nil {
		body["params"] = params
	}
	return body
}

func initializeParams() map[string]any {
	return map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "agenvoy",
			"version": "0.1.0",
		},
	}
}
