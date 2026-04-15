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

package endpoints

import (
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	apiPrefix = "/api/v1"

	AlertsPath          = apiPrefix + "/alerts"
	AlertManagersPath   = apiPrefix + "/alertmanagers"
	QueryPath           = apiPrefix + "/query"
	QueryRangePath      = apiPrefix + "/query_range"
	QueryExemplarsPath  = apiPrefix + "/query_exemplars"
	LabelsPath          = apiPrefix + "/labels"
	LabelValuesPath     = apiPrefix + "/label/:name/values"
	SeriesPath          = apiPrefix + "/series"
	TargetsPath         = apiPrefix + "/targets"
	TargetsMetadataPath = apiPrefix + "/targets/metadata"
	MetadataPath        = apiPrefix + "/metadata"
	RulesPath           = apiPrefix + "/rules"
	SnapshotPath        = apiPrefix + "/admin/tsdb/snapshot"
	DeleteSeriesPath    = apiPrefix + "/admin/tsdb/delete_series"
	CleanTombstonesPath = apiPrefix + "/admin/tsdb/clean_tombstones"
	ConfigPath          = apiPrefix + "/status/config"
	FlagsPath           = apiPrefix + "/status/flags"
	BuildinfoPath       = apiPrefix + "/status/buildinfo"
	RuntimeinfoPath     = apiPrefix + "/status/runtimeinfo"
	TSDBPath            = apiPrefix + "/status/tsdb"
	TSDBBlocksPath      = apiPrefix + "/status/tsdb/blocks"
	WalReplayPath       = apiPrefix + "/status/walreplay"
	FormatQueryPath     = apiPrefix + "/format_query"
)

type Options struct {
	Timeout       time.Duration
	LookbackDelta time.Duration
	Stats         string
	Limit         uint64
}

func NewRequest(method string, u *url.URL) (*http.Request, error) {
	return http.NewRequest(method, u.String(), nil)
}

func AddMatchers(q url.Values, matches []string) {
	for _, m := range matches {
		q.Add("match[]", m)
	}
}

func AddTimeRange(q url.Values, startTime, endTime time.Time) {
	if !startTime.IsZero() {
		q.Set("start", FormatTime(startTime))
	}
	if !endTime.IsZero() {
		q.Set("end", FormatTime(endTime))
	}
}

func AddOptionalURLParams(q url.Values, opts Options) url.Values {
	if opts.Timeout > 0 {
		q.Set("timeout", opts.Timeout.String())
	}
	if opts.LookbackDelta > 0 {
		q.Set("lookback_delta", opts.LookbackDelta.String())
	}
	if opts.Stats != "" {
		q.Set("stats", opts.Stats)
	}
	if opts.Limit > 0 {
		q.Set("limit", strconv.FormatUint(opts.Limit, 10))
	}
	return q
}

func FormatTime(t time.Time) string {
	return strconv.FormatFloat(float64(t.Unix())+float64(t.Nanosecond())/1e9, 'f', -1, 64)
}
