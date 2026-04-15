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

package codec

import (
	"math"
	"reflect"
	"testing"

	"github.com/prometheus/common/model"
)

func TestParseResponse_Success(t *testing.T) {
	body := []byte(`{"status":"success","data":{"resultType":"scalar","result":[1234567890,"42"]},"warnings":["w1"]}`)

	r, err := ParseResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Status != "success" {
		t.Errorf("expected status 'success', got %q", r.Status)
	}
	if len(r.Warnings) != 1 || r.Warnings[0] != "w1" {
		t.Errorf("expected warnings [w1], got %v", r.Warnings)
	}
	if r.ErrType != "" || r.ErrMsg != "" {
		t.Errorf("expected no error fields, got type=%q msg=%q", r.ErrType, r.ErrMsg)
	}
	if string(r.Data) != `{"resultType":"scalar","result":[1234567890,"42"]}` {
		t.Errorf("unexpected data: %s", r.Data)
	}
}

func TestParseResponse_APIError(t *testing.T) {
	body := []byte(`{"status":"error","errorType":"bad_data","error":"query parse error","data":null}`)

	r, err := ParseResponse(body)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if r.Status != "error" {
		t.Errorf("expected status 'error', got %q", r.Status)
	}
	if r.ErrType != "bad_data" {
		t.Errorf("expected errorType 'bad_data', got %q", r.ErrType)
	}
	if r.ErrMsg != "query parse error" {
		t.Errorf("expected error 'query parse error', got %q", r.ErrMsg)
	}
}

func TestParseResponse_MalformedJSON(t *testing.T) {
	body := []byte(`not valid json`)

	_, err := ParseResponse(body)
	if err == nil {
		t.Fatal("expected an error for malformed JSON, got nil")
	}
}

func TestParseResponse_NoWarnings(t *testing.T) {
	body := []byte(`{"status":"success","data":null}`)

	r, err := ParseResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Warnings != nil {
		t.Errorf("expected nil warnings, got %v", r.Warnings)
	}
}

func TestParseResponse_WithWarningsAndError(t *testing.T) {
	body := []byte(`{"status":"error","errorType":"timeout","error":"context deadline exceeded","warnings":["partial results","index out of bounds"]}`)

	r, err := ParseResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.ErrType != "timeout" {
		t.Errorf("expected ErrType 'timeout', got %q", r.ErrType)
	}
	if len(r.Warnings) != 2 {
		t.Errorf("expected 2 warnings, got %d: %v", len(r.Warnings), r.Warnings)
	}
	if r.Warnings[0] != "partial results" || r.Warnings[1] != "index out of bounds" {
		t.Errorf("unexpected warnings: %v", r.Warnings)
	}
}

// TestSamplePairCodecRegistered verifies that init() activated the custom
// encoder so that the wire format is [timestamp_float, "value_string"].
func TestSamplePairCodecRegistered(t *testing.T) {
	pair := model.SamplePair{Timestamp: 1000, Value: 42}
	b, err := JSON.Marshal(pair)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `[1,"42"]`
	if string(b) != want {
		t.Errorf("SamplePair wire format: got %s, want %s", b, want)
	}

	var got model.SamplePair
	if err := JSON.Unmarshal([]byte(want), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(pair, got) {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, pair)
	}
}

// TestSamplePairSpecialValues confirms NaN and Inf survive round-trip.
func TestSamplePairSpecialValues(t *testing.T) {
	cases := []model.SampleValue{
		model.SampleValue(math.NaN()),
		model.SampleValue(math.Inf(1)),
		model.SampleValue(math.Inf(-1)),
	}
	for _, v := range cases {
		pair := model.SamplePair{Timestamp: 0, Value: v}
		b, err := JSON.Marshal(pair)
		if err != nil {
			t.Fatalf("marshal %v: %v", v, err)
		}
		var got model.SamplePair
		if err := JSON.Unmarshal(b, &got); err != nil {
			t.Fatalf("unmarshal %v: %v", v, err)
		}
		if math.IsNaN(float64(v)) {
			if !math.IsNaN(float64(got.Value)) {
				t.Errorf("NaN round-trip failed: got %v", got.Value)
			}
		} else if float64(got.Value) != float64(v) {
			t.Errorf("round-trip %v: got %v", v, got.Value)
		}
	}
}
