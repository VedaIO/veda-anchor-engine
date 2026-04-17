package ipc

import (
	"encoding/json"
	"veda-anchor-engine/src/internal/config"
)

// Request is a message received from the client.
type Request struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// Response is a message sent to the client.
type Response struct {
	ID     string `json:"id"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// GetIPCAddress returns the Windows Named Pipe address.
func GetIPCAddress() string {
	return config.PipeName
}
