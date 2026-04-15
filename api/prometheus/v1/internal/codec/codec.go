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

// Package codec registers custom json-iterator encoders/decoders for
// Prometheus model types and provides the ParseResponse helper for parsing
// the standard Prometheus HTTP API response envelope.
//
// Importing this package as a side effect (import _ "...") is sufficient to
// activate the codec registrations for any code that uses the shared
// jsoniter.ConfigCompatibleWithStandardLibrary instance.
package codec

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
	"github.com/prometheus/common/model"
)

// JSON is the shared jsoniter API instance used by the Prometheus v1 client.
// It is identical to jsoniter.ConfigCompatibleWithStandardLibrary; exposing it
// here allows callers that need the registered type codecs to use a single
// well-known instance.
var JSON = jsoniter.ConfigCompatibleWithStandardLibrary

func init() {
	// Register using the package-level functions so the codecs apply to all
	// jsoniter API instances, not just the JSON variable above.
	jsoniter.RegisterTypeEncoderFunc("model.SamplePair", marshalSamplePairJSON, marshalJSONIsEmpty)
	jsoniter.RegisterTypeDecoderFunc("model.SamplePair", unmarshalSamplePairJSON)
	jsoniter.RegisterTypeEncoderFunc("model.SampleHistogramPair", marshalSampleHistogramPairJSON, marshalJSONIsEmpty)
	jsoniter.RegisterTypeDecoderFunc("model.SampleHistogramPair", unmarshalSampleHistogramPairJSON)
	jsoniter.RegisterTypeEncoderFunc("model.SampleStream", marshalSampleStreamJSON, marshalJSONIsEmpty) // Only needed for benchmark.
	jsoniter.RegisterTypeDecoderFunc("model.SampleStream", unmarshalSampleStreamJSON)                   // Only needed for benchmark.
}

// Response is the parsed Prometheus API response envelope.
// Data holds the raw JSON bytes of the "data" field; the caller is responsible
// for unmarshalling it into the appropriate result type.
type Response struct {
	Status   string
	Data     []byte
	Warnings []string
	// ErrType and ErrMsg are populated only when Status == "error".
	ErrType string
	ErrMsg  string
}

// ParseResponse parses a raw Prometheus HTTP API response envelope and
// returns the structured fields. It returns a non-nil error only when the
// JSON is malformed; API-level errors are surfaced via Response.Status/ErrType.
func ParseResponse(body []byte) (*Response, error) {
	var result struct {
		Status    string                `json:"status"`
		Data      jsoniter.RawMessage   `json:"data"`
		ErrorType string                `json:"errorType"`
		Error     string                `json:"error"`
		Warnings  []string              `json:"warnings,omitempty"`
	}
	if err := JSON.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &Response{
		Status:   result.Status,
		Data:     []byte(result.Data),
		Warnings: result.Warnings,
		ErrType:  result.ErrorType,
		ErrMsg:   result.Error,
	}, nil
}

func unmarshalSamplePairJSON(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
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

func marshalSamplePairJSON(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	p := *((*model.SamplePair)(ptr))
	stream.WriteArrayStart()
	marshalTimestamp(p.Timestamp, stream)
	stream.WriteMore()
	marshalFloat(float64(p.Value), stream)
	stream.WriteArrayEnd()
}

func unmarshalSampleHistogramPairJSON(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
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

func marshalSampleHistogramPairJSON(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	p := *((*model.SampleHistogramPair)(ptr))
	stream.WriteArrayStart()
	marshalTimestamp(p.Timestamp, stream)
	stream.WriteMore()
	marshalHistogram(*p.Histogram, stream)
	stream.WriteArrayEnd()
}

func unmarshalSampleStreamJSON(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	ss := (*model.SampleStream)(ptr)
	for key := iter.ReadObject(); key != ""; key = iter.ReadObject() {
		switch key {
		case "metric":
			metricString := iter.ReadAny().ToString()
			if err := JSON.UnmarshalFromString(metricString, &ss.Metric); err != nil {
				iter.ReportError("unmarshal model.SampleStream", err.Error())
				return
			}
		case "values":
			for iter.ReadArray() {
				v := model.SamplePair{}
				unmarshalSamplePairJSON(unsafe.Pointer(&v), iter)
				ss.Values = append(ss.Values, v)
			}
		case "histograms":
			for iter.ReadArray() {
				h := model.SampleHistogramPair{}
				unmarshalSampleHistogramPairJSON(unsafe.Pointer(&h), iter)
				ss.Histograms = append(ss.Histograms, h)
			}
		default:
			iter.ReportError("unmarshal model.SampleStream", fmt.Sprint("unexpected key:", key))
			return
		}
	}
}

func marshalSampleStreamJSON(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	ss := *((*model.SampleStream)(ptr))
	stream.WriteObjectStart()
	stream.WriteObjectField(`metric`)
	m, err := JSON.Marshal(ss.Metric)
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
			marshalSamplePairJSON(unsafe.Pointer(&v), stream)
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
			marshalSampleHistogramPairJSON(unsafe.Pointer(&h), stream)
		}
		stream.WriteArrayEnd()
	}
	stream.WriteObjectEnd()
}

func marshalFloat(v float64, stream *jsoniter.Stream) {
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

func marshalTimestamp(timestamp model.Time, stream *jsoniter.Stream) {
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

func unmarshalHistogramBucket(iter *jsoniter.Iterator) (*model.HistogramBucket, error) {
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

// marshalHistogramBucket writes something like: [ 3, "-0.25", "0.25", "3"]
// See marshalHistogram to understand what the numbers mean
func marshalHistogramBucket(b model.HistogramBucket, stream *jsoniter.Stream) {
	stream.WriteArrayStart()
	stream.WriteInt32(b.Boundaries)
	stream.WriteMore()
	marshalFloat(float64(b.Lower), stream)
	stream.WriteMore()
	marshalFloat(float64(b.Upper), stream)
	stream.WriteMore()
	marshalFloat(float64(b.Count), stream)
	stream.WriteArrayEnd()
}

// marshalHistogram writes something like:
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
// The 2nd and 3rd elements are the lower and upper boundary. The 4th element is
// the bucket count.
func marshalHistogram(h model.SampleHistogram, stream *jsoniter.Stream) {
	stream.WriteObjectStart()
	stream.WriteObjectField(`count`)
	marshalFloat(float64(h.Count), stream)
	stream.WriteMore()
	stream.WriteObjectField(`sum`)
	marshalFloat(float64(h.Sum), stream)

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
		marshalHistogramBucket(*bucket, stream)
	}
	if bucketFound {
		stream.WriteArrayEnd()
	}
	stream.WriteObjectEnd()
}

func marshalJSONIsEmpty(ptr unsafe.Pointer) bool {
	return false
}
