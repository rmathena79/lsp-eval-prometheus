// Copyright 2026 The Prometheus Authors
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
	"testing"

	json "github.com/json-iterator/go"

	"github.com/prometheus/common/model"
)

func TestCustomCodecRegistrationForSamplePair(t *testing.T) {
	pair := model.SamplePair{Timestamp: 1001, Value: model.SampleValue(math.Inf(1))}

	b, err := json.Marshal(pair)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `[1.001,"+Inf"]` {
		t.Fatalf("unexpected marshal output: %s", string(b))
	}

	var decoded model.SamplePair
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Timestamp != pair.Timestamp || decoded.Value != pair.Value {
		t.Fatalf("unexpected round trip: %#v", decoded)
	}
}

func TestCustomCodecRegistrationForSampleStream(t *testing.T) {
	stream := model.SampleStream{
		Metric: model.Metric{"__name__": "up", "job": "prometheus"},
		Values: []model.SamplePair{
			{Timestamp: 1001, Value: 2},
		},
	}

	b, err := json.Marshal(stream)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"metric":{"__name__":"up","job":"prometheus"},"values":[[1.001,"2"]]}`
	if string(b) != want {
		t.Fatalf("unexpected marshal output: want %s, got %s", want, string(b))
	}

	var decoded model.SampleStream
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Values) != 1 || decoded.Values[0].Timestamp != 1001 || decoded.Values[0].Value != 2 {
		t.Fatalf("unexpected round trip: %#v", decoded)
	}
}
