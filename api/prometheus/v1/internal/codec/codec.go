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
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"unsafe"

	json "github.com/json-iterator/go"

	internaltypes "github.com/prometheus/client_golang/api/prometheus/v1/internal/types"
	"github.com/prometheus/common/model"
)

var registerOnce sync.Once

func Register() {
	registerOnce.Do(func() {
		json.RegisterTypeEncoderFunc("model.SamplePair", marshalSamplePairJSON, marshalJSONIsEmpty)
		json.RegisterTypeDecoderFunc("model.SamplePair", unmarshalSamplePairJSON)
		json.RegisterTypeEncoderFunc("model.SampleHistogramPair", marshalSampleHistogramPairJSON, marshalJSONIsEmpty)
		json.RegisterTypeDecoderFunc("model.SampleHistogramPair", unmarshalSampleHistogramPairJSON)
		json.RegisterTypeEncoderFunc("model.SampleStream", marshalSampleStreamJSON, marshalJSONIsEmpty)
		json.RegisterTypeDecoderFunc("model.SampleStream", unmarshalSampleStreamJSON)
	})
}

func DecodeRuleGroupPayload(b []byte) (internaltypes.RuleGroupPayload, error) {
	var payload internaltypes.RuleGroupPayload
	return payload, json.Unmarshal(b, &payload)
}

func DecodeRuleType(b []byte) (string, error) {
	var payload internaltypes.RuleTypePayload
	if err := json.Unmarshal(b, &payload); err != nil {
		return "", err
	}
	if payload.Type == "" {
		return "", errors.New("type field not present in rule")
	}
	return payload.Type, nil
}

func DecodeAlertingRulePayload(b []byte) (internaltypes.AlertingRulePayload, error) {
	var payload internaltypes.AlertingRulePayload
	return payload, json.Unmarshal(b, &payload)
}

func DecodeRecordingRulePayload(b []byte) (internaltypes.RecordingRulePayload, error) {
	var payload internaltypes.RecordingRulePayload
	return payload, json.Unmarshal(b, &payload)
}

func DecodeQueryResult(b []byte) (model.Value, error) {
	var payload struct {
		Type   model.ValueType `json:"resultType"`
		Result json.RawMessage `json:"result"`
	}

	if err := json.Unmarshal(b, &payload); err != nil {
		return nil, err
	}

	switch payload.Type {
	case model.ValScalar:
		var value model.Scalar
		if err := json.Unmarshal(payload.Result, &value); err != nil {
			return nil, err
		}
		return &value, nil
	case model.ValVector:
		var value model.Vector
		if err := json.Unmarshal(payload.Result, &value); err != nil {
			return nil, err
		}
		return value, nil
	case model.ValMatrix:
		var value model.Matrix
		if err := json.Unmarshal(payload.Result, &value); err != nil {
			return nil, err
		}
		return value, nil
	default:
		return nil, fmt.Errorf("unexpected value type %q", payload.Type)
	}
}

func unmarshalSamplePairJSON(ptr unsafe.Pointer, iter *json.Iterator) {
	p := (*model.SamplePair)(ptr)
	if !iter.ReadArray() {
		iter.ReportError("unmarshal model.SamplePair", "SamplePair must be [timestamp, value]")
		return
	}
	t := iter.ReadNumber()
	if err := p.Timestamp.UnmarshalJSON([]byte(t)); err != nil {
		iter.ReportError("unmarshal model.SamplePair", err.Error())
		return
	}
	if !iter.ReadArray() {
		iter.ReportError("unmarshal model.SamplePair", "SamplePair missing value")
		return
	}

	f, err := strconv.ParseFloat(iter.ReadString(), 64)
	if err != nil {
		iter.ReportError("unmarshal model.SamplePair", err.Error())
		return
	}
	p.Value = model.SampleValue(f)

	if iter.ReadArray() {
		iter.ReportError("unmarshal model.SamplePair", "SamplePair has too many values, must be [timestamp, value]")
	}
}

func marshalSamplePairJSON(ptr unsafe.Pointer, stream *json.Stream) {
	p := *((*model.SamplePair)(ptr))
	stream.WriteArrayStart()
	marshalTimestamp(p.Timestamp, stream)
	stream.WriteMore()
	marshalFloat(float64(p.Value), stream)
	stream.WriteArrayEnd()
}

func unmarshalSampleHistogramPairJSON(ptr unsafe.Pointer, iter *json.Iterator) {
	p := (*model.SampleHistogramPair)(ptr)
	if !iter.ReadArray() {
		iter.ReportError("unmarshal model.SampleHistogramPair", "SampleHistogramPair must be [timestamp, {histogram}]")
		return
	}
	t := iter.ReadNumber()
	if err := p.Timestamp.UnmarshalJSON([]byte(t)); err != nil {
		iter.ReportError("unmarshal model.SampleHistogramPair", err.Error())
		return
	}
	if !iter.ReadArray() {
		iter.ReportError("unmarshal model.SampleHistogramPair", "SamplePair missing histogram")
		return
	}
	h := &model.SampleHistogram{}
	p.Histogram = h
	for key := iter.ReadObject(); key != ""; key = iter.ReadObject() {
		switch key {
		case "count":
			f, err := strconv.ParseFloat(iter.ReadString(), 64)
			if err != nil {
				iter.ReportError("unmarshal model.SampleHistogramPair", "count of histogram is not a float")
				return
			}
			h.Count = model.FloatString(f)
		case "sum":
			f, err := strconv.ParseFloat(iter.ReadString(), 64)
			if err != nil {
				iter.ReportError("unmarshal model.SampleHistogramPair", "sum of histogram is not a float")
				return
			}
			h.Sum = model.FloatString(f)
		case "buckets":
			for iter.ReadArray() {
				bucket, err := unmarshalHistogramBucket(iter)
				if err != nil {
					iter.ReportError("unmarshal model.HistogramBucket", err.Error())
					return
				}
				h.Buckets = append(h.Buckets, bucket)
			}
		default:
			iter.ReportError("unmarshal model.SampleHistogramPair", fmt.Sprint("unexpected key in histogram:", key))
			return
		}
	}
	if iter.ReadArray() {
		iter.ReportError("unmarshal model.SampleHistogramPair", "SampleHistogramPair has too many values, must be [timestamp, {histogram}]")
	}
}

func marshalSampleHistogramPairJSON(ptr unsafe.Pointer, stream *json.Stream) {
	p := *((*model.SampleHistogramPair)(ptr))
	stream.WriteArrayStart()
	marshalTimestamp(p.Timestamp, stream)
	stream.WriteMore()
	marshalHistogram(*p.Histogram, stream)
	stream.WriteArrayEnd()
}

func unmarshalSampleStreamJSON(ptr unsafe.Pointer, iter *json.Iterator) {
	ss := (*model.SampleStream)(ptr)
	for key := iter.ReadObject(); key != ""; key = iter.ReadObject() {
		switch key {
		case "metric":
			metricString := iter.ReadAny().ToString()
			if err := json.UnmarshalFromString(metricString, &ss.Metric); err != nil {
				iter.ReportError("unmarshal model.SampleStream", err.Error())
				return
			}
		case "values":
			for iter.ReadArray() {
				value := model.SamplePair{}
				unmarshalSamplePairJSON(unsafe.Pointer(&value), iter)
				ss.Values = append(ss.Values, value)
			}
		case "histograms":
			for iter.ReadArray() {
				histogram := model.SampleHistogramPair{}
				unmarshalSampleHistogramPairJSON(unsafe.Pointer(&histogram), iter)
				ss.Histograms = append(ss.Histograms, histogram)
			}
		default:
			iter.ReportError("unmarshal model.SampleStream", fmt.Sprint("unexpected key:", key))
			return
		}
	}
}

func marshalSampleStreamJSON(ptr unsafe.Pointer, stream *json.Stream) {
	ss := *((*model.SampleStream)(ptr))
	stream.WriteObjectStart()
	stream.WriteObjectField(`metric`)
	metricJSON, err := json.ConfigCompatibleWithStandardLibrary.Marshal(ss.Metric)
	if err != nil {
		stream.Error = err
		return
	}
	stream.SetBuffer(append(stream.Buffer(), metricJSON...))
	if len(ss.Values) > 0 {
		stream.WriteMore()
		stream.WriteObjectField(`values`)
		stream.WriteArrayStart()
		for i, value := range ss.Values {
			if i > 0 {
				stream.WriteMore()
			}
			marshalSamplePairJSON(unsafe.Pointer(&value), stream)
		}
		stream.WriteArrayEnd()
	}
	if len(ss.Histograms) > 0 {
		stream.WriteMore()
		stream.WriteObjectField(`histograms`)
		stream.WriteArrayStart()
		for i, histogram := range ss.Histograms {
			if i > 0 {
				stream.WriteMore()
			}
			marshalSampleHistogramPairJSON(unsafe.Pointer(&histogram), stream)
		}
		stream.WriteArrayEnd()
	}
	stream.WriteObjectEnd()
}

func marshalFloat(v float64, stream *json.Stream) {
	stream.WriteRaw(`"`)
	buf := stream.Buffer()
	abs := math.Abs(v)
	format := byte('f')
	if abs != 0 {
		if abs < 1e-6 || abs >= 1e21 {
			format = 'e'
		}
	}
	buf = strconv.AppendFloat(buf, v, format, -1, 64)
	stream.SetBuffer(buf)
	stream.WriteRaw(`"`)
}

func marshalTimestamp(timestamp model.Time, stream *json.Stream) {
	t := int64(timestamp)
	if t < 0 {
		stream.WriteRaw(`-`)
		t = -t
	}
	stream.WriteInt64(t / 1000)
	fraction := t % 1000
	if fraction != 0 {
		stream.WriteRaw(`.`)
		if fraction < 100 {
			stream.WriteRaw(`0`)
		}
		if fraction < 10 {
			stream.WriteRaw(`0`)
		}
		stream.WriteInt64(fraction)
	}
}

func unmarshalHistogramBucket(iter *json.Iterator) (*model.HistogramBucket, error) {
	bucket := model.HistogramBucket{}
	if !iter.ReadArray() {
		return nil, errors.New("HistogramBucket must be [boundaries, lower, upper, count]")
	}
	boundaries, err := iter.ReadNumber().Int64()
	if err != nil {
		return nil, err
	}
	bucket.Boundaries = int32(boundaries)
	if !iter.ReadArray() {
		return nil, errors.New("HistogramBucket must be [boundaries, lower, upper, count]")
	}
	f, err := strconv.ParseFloat(iter.ReadString(), 64)
	if err != nil {
		return nil, err
	}
	bucket.Lower = model.FloatString(f)
	if !iter.ReadArray() {
		return nil, errors.New("HistogramBucket must be [boundaries, lower, upper, count]")
	}
	f, err = strconv.ParseFloat(iter.ReadString(), 64)
	if err != nil {
		return nil, err
	}
	bucket.Upper = model.FloatString(f)
	if !iter.ReadArray() {
		return nil, errors.New("HistogramBucket must be [boundaries, lower, upper, count]")
	}
	f, err = strconv.ParseFloat(iter.ReadString(), 64)
	if err != nil {
		return nil, err
	}
	bucket.Count = model.FloatString(f)
	if iter.ReadArray() {
		return nil, errors.New("HistogramBucket has too many values, must be [boundaries, lower, upper, count]")
	}
	return &bucket, nil
}

func marshalHistogramBucket(bucket model.HistogramBucket, stream *json.Stream) {
	stream.WriteArrayStart()
	stream.WriteInt32(bucket.Boundaries)
	stream.WriteMore()
	marshalFloat(float64(bucket.Lower), stream)
	stream.WriteMore()
	marshalFloat(float64(bucket.Upper), stream)
	stream.WriteMore()
	marshalFloat(float64(bucket.Count), stream)
	stream.WriteArrayEnd()
}

func marshalHistogram(histogram model.SampleHistogram, stream *json.Stream) {
	stream.WriteObjectStart()
	stream.WriteObjectField(`count`)
	marshalFloat(float64(histogram.Count), stream)
	stream.WriteMore()
	stream.WriteObjectField(`sum`)
	marshalFloat(float64(histogram.Sum), stream)

	bucketFound := false
	for _, bucket := range histogram.Buckets {
		if bucket.Count == 0 {
			continue
		}
		stream.WriteMore()
		if !bucketFound {
			stream.WriteObjectField(`buckets`)
			stream.WriteArrayStart()
		}
		bucketFound = true
		marshalHistogramBucket(*bucket, stream)
	}
	if bucketFound {
		stream.WriteArrayEnd()
	}
	stream.WriteObjectEnd()
}

func marshalJSONIsEmpty(unsafe.Pointer) bool {
	return false
}
