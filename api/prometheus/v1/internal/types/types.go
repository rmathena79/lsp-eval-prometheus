package types

import (
	json "github.com/json-iterator/go"

	"github.com/prometheus/common/model"
)

type APIResponse struct {
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data"`
	ErrorType string          `json:"errorType"`
	Error     string          `json:"error"`
	Warnings  []string        `json:"warnings,omitempty"`
}

type QueryResult struct {
	Type   model.ValueType `json:"resultType"`
	Result json.RawMessage `json:"result"`
}
