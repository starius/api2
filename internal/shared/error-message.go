package shared

import "encoding/json"

type ErrorMessage struct {
	Error  string          `json:"error"`
	Detail json.RawMessage `json:"detail,omitempty"`
	Code   string          `json:"code,omitempty"`
}
