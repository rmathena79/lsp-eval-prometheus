// Copyright 2017 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	"time"

	json "github.com/json-iterator/go"

	"github.com/prometheus/common/model"
)

// APIOptions collects optional query parameters supported by several v1 endpoints.
type APIOptions struct {
	Timeout       time.Duration
	LookbackDelta time.Duration
	Stats         string
	Limit         uint64
}

// APIResponse models the Prometheus API response envelope.
type APIResponse struct {
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data"`
	ErrorType string          `json:"errorType"`
	Error     string          `json:"error"`
	Warnings  []string        `json:"warnings,omitempty"`
}

// RuleGroupPayload is the intermediate JSON representation for a rule group.
type RuleGroupPayload struct {
	Name     string            `json:"name"`
	File     string            `json:"file"`
	Interval float64           `json:"interval"`
	Rules    []json.RawMessage `json:"rules"`
}

// RuleTypePayload is the minimal rule shape used to validate type tags.
type RuleTypePayload struct {
	Type string `json:"type"`
}

// AlertingRulePayload is the intermediate JSON representation for an alerting rule.
type AlertingRulePayload struct {
	Name           string          `json:"name"`
	Query          string          `json:"query"`
	Duration       float64         `json:"duration"`
	Labels         model.LabelSet  `json:"labels"`
	Annotations    model.LabelSet  `json:"annotations"`
	Alerts         json.RawMessage `json:"alerts"`
	Health         string          `json:"health"`
	LastError      string          `json:"lastError,omitempty"`
	EvaluationTime float64         `json:"evaluationTime"`
	LastEvaluation time.Time       `json:"lastEvaluation"`
	State          string          `json:"state"`
}

// RecordingRulePayload is the intermediate JSON representation for a recording rule.
type RecordingRulePayload struct {
	Name           string         `json:"name"`
	Query          string         `json:"query"`
	Labels         model.LabelSet `json:"labels,omitempty"`
	Health         string         `json:"health"`
	LastError      string         `json:"lastError,omitempty"`
	EvaluationTime float64        `json:"evaluationTime"`
	LastEvaluation time.Time      `json:"lastEvaluation"`
}
