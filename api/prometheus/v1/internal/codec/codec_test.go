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

package codec_test

import (
	"math"
	"testing"

	json "github.com/json-iterator/go"

	"github.com/prometheus/common/model"

	"github.com/prometheus/client_golang/api/prometheus/v1/internal/codec"
)

// ---------------------------------------------------------------------------
// Custom codec registration — SamplePair round-trip
// ---------------------------------------------------------------------------

func TestSamplePairCodecRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		pair     model.SamplePair
		wantJSON string
	}{
		{"zero", model.SamplePair{Timestamp: 0, Value: 0}, `[0,"0"]`},
		{"positive", model.SamplePair{Timestamp: 1001, Value: 20}, `[1.001,"20"]`},
		{"negative ts", model.SamplePair{Timestamp: -1, Value: 20}, `[-0.001,"20"]`},
		{"NaN", model.SamplePair{Timestamp: 0, Value: model.SampleValue(math.NaN())}, `[0,"NaN"]`},
		{"+Inf", model.SamplePair{Timestamp: 0, Value: model.SampleValue(math.Inf(1))}, `[0,"+Inf"]`},
		{"-Inf", model.SamplePair{Timestamp: 0, Value: model.SampleValue(math.Inf(-1))}, `[0,"-Inf"]`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.pair)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(b) != tc.wantJSON {
				t.Errorf("marshal: want %s, got %s", tc.wantJSON, string(b))
			}

			// Unmarshal back and re-marshal so NaN equality works via string compare.
			var got model.SamplePair
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			b2, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("re-marshal: %v", err)
			}
			if string(b2) != tc.wantJSON {
				t.Errorf("roundtrip: want %s, got %s", tc.wantJSON, string(b2))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Custom codec registration — SampleHistogramPair round-trip
// ---------------------------------------------------------------------------

func TestSampleHistogramPairCodecRoundTrip(t *testing.T) {
	pair := model.SampleHistogramPair{
		Timestamp: 1,
		Histogram: &model.SampleHistogram{
			Count: 13.5,
			Sum:   3897.1,
			Buckets: model.HistogramBuckets{
				{Boundaries: 1, Lower: -4870.992343051145, Upper: -4466.7196729968955, Count: 1},
				{Boundaries: 0, Lower: 2048, Upper: 2233.3598364984477, Count: 1.5},
			},
		},
	}

	b, err := json.Marshal(pair)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got model.SampleHistogramPair
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	b2, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}

	if string(b) != string(b2) {
		t.Errorf("roundtrip mismatch:\n  want: %s\n  got:  %s", b, b2)
	}
}

func TestSampleHistogramPairEmptyBuckets(t *testing.T) {
	// Empty buckets must not appear in the JSON output.
	pair := model.SampleHistogramPair{
		Timestamp: 0,
		Histogram: &model.SampleHistogram{
			Count: 5,
			Sum:   10,
			Buckets: model.HistogramBuckets{
				{Boundaries: 0, Lower: 0, Upper: 1, Count: 0},  // zero count — omitted
				{Boundaries: 0, Lower: 1, Upper: 2, Count: 5},  // non-zero — included
			},
		},
	}

	b, err := json.Marshal(pair)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got model.SampleHistogramPair
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// After round-trip, only the non-zero bucket should be present.
	if len(got.Histogram.Buckets) != 1 {
		t.Errorf("expected 1 bucket after omitting zero-count, got %d", len(got.Histogram.Buckets))
	}
}

// ---------------------------------------------------------------------------
// QueryResult.UnmarshalJSON
// ---------------------------------------------------------------------------

func TestQueryResultUnmarshalScalar(t *testing.T) {
	raw := `{"resultType":"scalar","result":[1677587274.055,"42"]}`

	var qr codec.QueryResult
	if err := json.Unmarshal([]byte(raw), &qr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if qr.V == nil {
		t.Fatal("expected non-nil V")
	}
	if qr.Type != model.ValScalar {
		t.Errorf("expected ValScalar, got %s", qr.Type)
	}
	sv, ok := qr.V.(*model.Scalar)
	if !ok {
		t.Fatalf("expected *model.Scalar, got %T", qr.V)
	}
	if sv.Value != 42 {
		t.Errorf("expected value 42, got %v", sv.Value)
	}
}

func TestQueryResultUnmarshalVector(t *testing.T) {
	raw := `{"resultType":"vector","result":[{"metric":{"job":"prom"},"value":[1677587274.055,"1"]}]}`

	var qr codec.QueryResult
	if err := json.Unmarshal([]byte(raw), &qr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if qr.Type != model.ValVector {
		t.Errorf("expected ValVector, got %s", qr.Type)
	}
	vv, ok := qr.V.(model.Vector)
	if !ok {
		t.Fatalf("expected model.Vector, got %T", qr.V)
	}
	if len(vv) != 1 {
		t.Errorf("expected 1 sample, got %d", len(vv))
	}
}

func TestQueryResultUnmarshalMatrix(t *testing.T) {
	raw := `{"resultType":"matrix","result":[{"metric":{"job":"prom"},"values":[[1677587274.055,"1"]]}]}`

	var qr codec.QueryResult
	if err := json.Unmarshal([]byte(raw), &qr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if qr.Type != model.ValMatrix {
		t.Errorf("expected ValMatrix, got %s", qr.Type)
	}
	mv, ok := qr.V.(model.Matrix)
	if !ok {
		t.Fatalf("expected model.Matrix, got %T", qr.V)
	}
	if len(mv) != 1 {
		t.Errorf("expected 1 stream, got %d", len(mv))
	}
}

func TestQueryResultUnmarshalUnknownType(t *testing.T) {
	raw := `{"resultType":"unknown","result":[]}`

	var qr codec.QueryResult
	if err := json.Unmarshal([]byte(raw), &qr); err == nil {
		t.Fatal("expected error for unknown resultType, got nil")
	}
}

// ---------------------------------------------------------------------------
// MarshalFloat edge cases
// ---------------------------------------------------------------------------

func TestMarshalFloatSpecialValues(t *testing.T) {
	// These are handled by the SamplePair codec so testing through it is cleanest.
	tests := []struct {
		value    model.SampleValue
		wantJSON string
	}{
		{model.SampleValue(math.NaN()), `[0,"NaN"]`},
		{model.SampleValue(math.Inf(1)), `[0,"+Inf"]`},
		{model.SampleValue(math.Inf(-1)), `[0,"-Inf"]`},
		{model.SampleValue(1e-7), `[0,"1e-07"]`},  // scientific notation for small values
		{model.SampleValue(1e22), `[0,"1e+22"]`},  // scientific notation for large values
	}

	for _, tc := range tests {
		pair := model.SamplePair{Timestamp: 0, Value: tc.value}
		b, err := json.Marshal(pair)
		if err != nil {
			t.Fatalf("marshal %v: %v", tc.value, err)
		}
		if string(b) != tc.wantJSON {
			t.Errorf("marshal %v: want %s, got %s", tc.value, tc.wantJSON, string(b))
		}
	}
}
