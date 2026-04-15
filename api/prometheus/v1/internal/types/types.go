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
	"fmt"

	json "github.com/json-iterator/go"

	"github.com/prometheus/common/model"
)

type ErrorType string

const (
	ErrBadResponse ErrorType = "bad_response"
	ErrServer      ErrorType = "server_error"
	ErrClient      ErrorType = "client_error"
)

type Error struct {
	Type   ErrorType
	Msg    string
	Detail string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Msg)
}

type APIResponse struct {
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data"`
	ErrorType ErrorType       `json:"errorType"`
	Error     string          `json:"error"`
	Warnings  []string        `json:"warnings,omitempty"`
}

type QueryResult struct {
	Type   model.ValueType `json:"resultType"`
	Result interface{}     `json:"result"`
	V      model.Value     `json:"-"`
}

func (qr *QueryResult) Value() model.Value {
	return qr.V
}

func (qr *QueryResult) UnmarshalJSON(b []byte) error {
	v := struct {
		Type   model.ValueType `json:"resultType"`
		Result json.RawMessage `json:"result"`
	}{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch v.Type {
	case model.ValScalar:
		var sv model.Scalar
		if err := json.Unmarshal(v.Result, &sv); err != nil {
			return err
		}
		qr.V = &sv
	case model.ValVector:
		var vv model.Vector
		if err := json.Unmarshal(v.Result, &vv); err != nil {
			return err
		}
		qr.V = vv
	case model.ValMatrix:
		var mv model.Matrix
		if err := json.Unmarshal(v.Result, &mv); err != nil {
			return err
		}
		qr.V = mv
	default:
		return fmt.Errorf("unexpected value type %q", v.Type)
	}
	qr.Type = v.Type
	return nil
}
