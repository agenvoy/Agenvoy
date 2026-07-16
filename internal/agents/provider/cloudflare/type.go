package cloudflare

import "github.com/pardnchiu/agenvoy/internal/agents/provider"

type response struct {
	Result  provider.Output `json:"result"`
	Success bool            `json:"success"`
	Errors  []struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"errors"`
}
