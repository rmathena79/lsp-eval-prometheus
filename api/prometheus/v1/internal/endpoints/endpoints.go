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
	"net/url"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/api/prometheus/v1/internal/types"
)

const (
	APIPrefix = "/api/v1"

	Alerts          = APIPrefix + "/alerts"
	AlertManagers   = APIPrefix + "/alertmanagers"
	Query           = APIPrefix + "/query"
	QueryRange      = APIPrefix + "/query_range"
	QueryExemplars  = APIPrefix + "/query_exemplars"
	Labels          = APIPrefix + "/labels"
	LabelValues     = APIPrefix + "/label/:name/values"
	Series          = APIPrefix + "/series"
	Targets         = APIPrefix + "/targets"
	TargetsMetadata = APIPrefix + "/targets/metadata"
	Metadata        = APIPrefix + "/metadata"
	Rules           = APIPrefix + "/rules"
	Snapshot        = APIPrefix + "/admin/tsdb/snapshot"
	DeleteSeries    = APIPrefix + "/admin/tsdb/delete_series"
	CleanTombstones = APIPrefix + "/admin/tsdb/clean_tombstones"
	Config          = APIPrefix + "/status/config"
	Flags           = APIPrefix + "/status/flags"
	Buildinfo       = APIPrefix + "/status/buildinfo"
	Runtimeinfo     = APIPrefix + "/status/runtimeinfo"
	TSDB            = APIPrefix + "/status/tsdb"
	TSDBBlocks      = APIPrefix + "/status/tsdb/blocks"
	WalReplay       = APIPrefix + "/status/walreplay"
	FormatQuery     = APIPrefix + "/format_query"
)

func ApplyOptions(q url.Values, opts types.APIOptions) url.Values {
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

func AddMatchers(q url.Values, matches []string) {
	for _, match := range matches {
		q.Add("match[]", match)
	}
}

func SetTime(q url.Values, name string, ts time.Time) {
	if !ts.IsZero() {
		q.Set(name, FormatTime(ts))
	}
}

func FormatTime(t time.Time) string {
	return strconv.FormatFloat(float64(t.Unix())+float64(t.Nanosecond())/1e9, 'f', -1, 64)
}
