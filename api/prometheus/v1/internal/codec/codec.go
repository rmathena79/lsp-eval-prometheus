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

// Package codec registers custom json-iterator encoder/decoder functions for
// Prometheus model types and exposes QueryResult, the internal type used to
// decode query responses.
//
// The init function here is the single place where json-iterator codec
// registration occurs for the entire v1 client. It runs automatically when
// the parent v1 package is imported (v1 imports this package).
package codec

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"unsafe"

	json "github.com/json-iterator/go"

	"github.com/prometheus/common/model"
)

func init() {
	json.RegisterTypeEncoderFunc("model.SamplePair", MarshalSamplePairJSON, MarshalJSONIsEmpty)
	json.RegisterTypeDecoderFunc("model.SamplePair", UnmarshalSamplePairJSON)
	json.RegisterTypeEncoderFunc("model.SampleHistogramPair", MarshalSampleHistogramPairJSON, MarshalJSONIsEmpty)
	json.RegisterTypeDecoderFunc("model.SampleHistogramPair", UnmarshalSampleHistogramPairJSON)
	json.RegisterTypeEncoderFunc("model.SampleStream", MarshalSampleStreamJSON, MarshalJSONIsEmpty) // Only needed for benchmark.
	json.RegisterTypeDecoderFunc("model.SampleStream", UnmarshalSampleStreamJSON)                   // Only needed for benchmark.
}

// QueryResult is the internal type used to decode a Prometheus query result
// envelope. Type and Result are populated from JSON; V holds the decoded
// model.Value after UnmarshalJSON runs.
type QueryResult struct {
	Type   model.ValueType `json:"resultType"`
	Result interface{}     `json:"result"`

	// V is the decoded model.Value, populated by UnmarshalJSON.
	V model.Value
}

// UnmarshalJSON implements json.Unmarshaler for QueryResult. It decodes the
// "resultType" discriminator and deserialises the "result" field into the
// appropriate model.Value concrete type.
func (qr *QueryResult) UnmarshalJSON(b []byte) error {
	v := struct {
		Type   model.ValueType `json:"resultType"`
		Result json.RawMessage `json:"result"`
	}{}

	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}

	qr.Type = v.Type
	switch v.Type {
	case model.ValScalar:
		var sv model.Scalar
		err = json.Unmarshal(v.Result, &sv)
		qr.V = &sv

	case model.ValVector:
		var vv model.Vector
		err = json.Unmarshal(v.Result, &vv)
		qr.V = vv

	case model.ValMatrix:
		var mv model.Matrix
		err = json.Unmarshal(v.Result, &mv)
		qr.V = mv

	default:
		err = fmt.Errorf("unexpected value type %q", v.Type)
	}
	return err
}

// UnmarshalSamplePairJSON is the json-iterator decoder for model.SamplePair.
func UnmarshalSamplePairJSON(ptr unsafe.Pointer, iter *json.Iterator) {
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
		return
	}
}

// MarshalSamplePairJSON is the json-iterator encoder for model.SamplePair.
func MarshalSamplePairJSON(ptr unsafe.Pointer, stream *json.Stream) {
	p := *((*model.SamplePair)(ptr))
	stream.WriteArrayStart()
	MarshalTimestamp(p.Timestamp, stream)
	stream.WriteMore()
	MarshalFloat(float64(p.Value), stream)
	stream.WriteArrayEnd()
}

// UnmarshalSampleHistogramPairJSON is the json-iterator decoder for
// model.SampleHistogramPair.
func UnmarshalSampleHistogramPairJSON(ptr unsafe.Pointer, iter *json.Iterator) {
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
				b, err := unmarshalHistogramBucket(iter)
				if err != nil {
					iter.ReportError("unmarshal model.HistogramBucket", err.Error())
					return
				}
				h.Buckets = append(h.Buckets, b)
			}
		default:
			iter.ReportError("unmarshal model.SampleHistogramPair", fmt.Sprint("unexpected key in histogram:", key))
			return
		}
	}
	if iter.ReadArray() {
		iter.ReportError("unmarshal model.SampleHistogramPair", "SampleHistogramPair has too many values, must be [timestamp, {histogram}]")
		return
	}
}

// MarshalSampleHistogramPairJSON is the json-iterator encoder for
// model.SampleHistogramPair.
func MarshalSampleHistogramPairJSON(ptr unsafe.Pointer, stream *json.Stream) {
	p := *((*model.SampleHistogramPair)(ptr))
	stream.WriteArrayStart()
	MarshalTimestamp(p.Timestamp, stream)
	stream.WriteMore()
	MarshalHistogram(*p.Histogram, stream)
	stream.WriteArrayEnd()
}

// UnmarshalSampleStreamJSON is the json-iterator decoder for model.SampleStream.
// Only needed for the benchmark.
func UnmarshalSampleStreamJSON(ptr unsafe.Pointer, iter *json.Iterator) {
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
				v := model.SamplePair{}
				UnmarshalSamplePairJSON(unsafe.Pointer(&v), iter)
				ss.Values = append(ss.Values, v)
			}
		case "histograms":
			for iter.ReadArray() {
				h := model.SampleHistogramPair{}
				UnmarshalSampleHistogramPairJSON(unsafe.Pointer(&h), iter)
				ss.Histograms = append(ss.Histograms, h)
			}
		default:
			iter.ReportError("unmarshal model.SampleStream", fmt.Sprint("unexpected key:", key))
			return
		}
	}
}

// MarshalSampleStreamJSON is the json-iterator encoder for model.SampleStream.
// Only needed for the benchmark.
func MarshalSampleStreamJSON(ptr unsafe.Pointer, stream *json.Stream) {
	ss := *((*model.SampleStream)(ptr))
	stream.WriteObjectStart()
	stream.WriteObjectField(`metric`)
	m, err := json.ConfigCompatibleWithStandardLibrary.Marshal(ss.Metric)
	if err != nil {
		stream.Error = err
		return
	}
	stream.SetBuffer(append(stream.Buffer(), m...))
	if len(ss.Values) > 0 {
		stream.WriteMore()
		stream.WriteObjectField(`values`)
		stream.WriteArrayStart()
		for i, v := range ss.Values {
			if i > 0 {
				stream.WriteMore()
			}
			MarshalSamplePairJSON(unsafe.Pointer(&v), stream)
		}
		stream.WriteArrayEnd()
	}
	if len(ss.Histograms) > 0 {
		stream.WriteMore()
		stream.WriteObjectField(`histograms`)
		stream.WriteArrayStart()
		for i, h := range ss.Histograms {
			if i > 0 {
				stream.WriteMore()
			}
			MarshalSampleHistogramPairJSON(unsafe.Pointer(&h), stream)
		}
		stream.WriteArrayEnd()
	}
	stream.WriteObjectEnd()
}

// MarshalFloat writes a float64 as a quoted JSON string, handling Inf and NaN
// which standard JSON does not permit as bare numbers.
func MarshalFloat(v float64, stream *json.Stream) {
	stream.WriteRaw(`"`)
	// Taken from https://github.com/json-iterator/go/blob/master/stream_float.go#L71 as a workaround
	// to https://github.com/json-iterator/go/issues/365 (json-iterator, to follow json standard, doesn't allow inf/nan).
	buf := stream.Buffer()
	abs := math.Abs(v)
	fmt := byte('f')
	// Note: Must use float32 comparisons for underlying float32 value to get precise cutoffs right.
	if abs != 0 {
		if abs < 1e-6 || abs >= 1e21 {
			fmt = 'e'
		}
	}
	buf = strconv.AppendFloat(buf, v, fmt, -1, 64)
	stream.SetBuffer(buf)
	stream.WriteRaw(`"`)
}

// MarshalTimestamp writes a model.Time as a decimal seconds timestamp, which
// is ~3x faster than converting to float64.
func MarshalTimestamp(timestamp model.Time, stream *json.Stream) {
	t := int64(timestamp)
	// Write out the timestamp as a float divided by 1000.
	// This is ~3x faster than converting to a float.
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

// unmarshalHistogramBucket decodes a single histogram bucket from the iterator.
func unmarshalHistogramBucket(iter *json.Iterator) (*model.HistogramBucket, error) {
	b := model.HistogramBucket{}
	if !iter.ReadArray() {
		return nil, errors.New("HistogramBucket must be [boundaries, lower, upper, count]")
	}
	boundaries, err := iter.ReadNumber().Int64()
	if err != nil {
		return nil, err
	}
	b.Boundaries = int32(boundaries)
	if !iter.ReadArray() {
		return nil, errors.New("HistogramBucket must be [boundaries, lower, upper, count]")
	}
	f, err := strconv.ParseFloat(iter.ReadString(), 64)
	if err != nil {
		return nil, err
	}
	b.Lower = model.FloatString(f)
	if !iter.ReadArray() {
		return nil, errors.New("HistogramBucket must be [boundaries, lower, upper, count]")
	}
	f, err = strconv.ParseFloat(iter.ReadString(), 64)
	if err != nil {
		return nil, err
	}
	b.Upper = model.FloatString(f)
	if !iter.ReadArray() {
		return nil, errors.New("HistogramBucket must be [boundaries, lower, upper, count]")
	}
	f, err = strconv.ParseFloat(iter.ReadString(), 64)
	if err != nil {
		return nil, err
	}
	b.Count = model.FloatString(f)
	if iter.ReadArray() {
		return nil, errors.New("HistogramBucket has too many values, must be [boundaries, lower, upper, count]")
	}
	return &b, nil
}

// MarshalHistogramBucket writes something like: [ 3, "-0.25", "0.25", "3"]
// See MarshalHistogram to understand what the numbers mean.
func MarshalHistogramBucket(b model.HistogramBucket, stream *json.Stream) {
	stream.WriteArrayStart()
	stream.WriteInt32(b.Boundaries)
	stream.WriteMore()
	MarshalFloat(float64(b.Lower), stream)
	stream.WriteMore()
	MarshalFloat(float64(b.Upper), stream)
	stream.WriteMore()
	MarshalFloat(float64(b.Count), stream)
	stream.WriteArrayEnd()
}

// MarshalHistogram writes a SampleHistogram as a JSON object.
//
// Example output:
//
//	{
//	    "count": "42",
//	    "sum": "34593.34",
//	    "buckets": [
//	      [ 3, "-0.25", "0.25", "3"],
//	      [ 0, "0.25", "0.5", "12"],
//	      [ 0, "0.5", "1", "21"],
//	      [ 0, "2", "4", "6"]
//	    ]
//	}
//
// The 1st element in each bucket array determines if the boundaries are
// inclusive (AKA closed) or exclusive (AKA open):
//
//	0: lower exclusive, upper inclusive
//	1: lower inclusive, upper exclusive
//	2: both exclusive
//	3: both inclusive
//
// The 2nd and 3rd elements are the lower and upper boundary. The 4th element
// is the bucket count.
func MarshalHistogram(h model.SampleHistogram, stream *json.Stream) {
	stream.WriteObjectStart()
	stream.WriteObjectField(`count`)
	MarshalFloat(float64(h.Count), stream)
	stream.WriteMore()
	stream.WriteObjectField(`sum`)
	MarshalFloat(float64(h.Sum), stream)

	bucketFound := false
	for _, bucket := range h.Buckets {
		if bucket.Count == 0 {
			continue // No need to expose empty buckets in JSON.
		}
		stream.WriteMore()
		if !bucketFound {
			stream.WriteObjectField(`buckets`)
			stream.WriteArrayStart()
		}
		bucketFound = true
		MarshalHistogramBucket(*bucket, stream)
	}
	if bucketFound {
		stream.WriteArrayEnd()
	}
	stream.WriteObjectEnd()
}

// MarshalJSONIsEmpty is a no-op isEmpty function for json-iterator type
// encoders; it always returns false so values are never omitted.
func MarshalJSONIsEmpty(ptr unsafe.Pointer) bool {
	return false
}
