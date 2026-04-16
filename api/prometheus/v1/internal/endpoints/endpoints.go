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

// Package endpoints defines the Prometheus v1 API endpoint path constants and
// shared request-construction helpers used by the httpAPI implementation.
package endpoints

import (
	"net/url"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/api/prometheus/v1/internal/types"
)

// API prefix and individual endpoint paths.
const (
	APIPrefix = "/api/v1"

	EPAlerts          = APIPrefix + "/alerts"
	EPAlertManagers   = APIPrefix + "/alertmanagers"
	EPQuery           = APIPrefix + "/query"
	EPQueryRange      = APIPrefix + "/query_range"
	EPQueryExemplars  = APIPrefix + "/query_exemplars"
	EPLabels          = APIPrefix + "/labels"
	EPLabelValues     = APIPrefix + "/label/:name/values"
	EPSeries          = APIPrefix + "/series"
	EPTargets         = APIPrefix + "/targets"
	EPTargetsMetadata = APIPrefix + "/targets/metadata"
	EPMetadata        = APIPrefix + "/metadata"
	EPRules           = APIPrefix + "/rules"
	EPSnapshot        = APIPrefix + "/admin/tsdb/snapshot"
	EPDeleteSeries    = APIPrefix + "/admin/tsdb/delete_series"
	EPCleanTombstones = APIPrefix + "/admin/tsdb/clean_tombstones"
	EPConfig          = APIPrefix + "/status/config"
	EPFlags           = APIPrefix + "/status/flags"
	EPBuildinfo       = APIPrefix + "/status/buildinfo"
	EPRuntimeinfo     = APIPrefix + "/status/runtimeinfo"
	EPTSDB            = APIPrefix + "/status/tsdb"
	EPTSDBBlocks      = APIPrefix + "/status/tsdb/blocks"
	EPWalReplay       = APIPrefix + "/status/walreplay"
	EPFormatQuery     = APIPrefix + "/format_query"
)

// AddOptionalURLParams applies any Option values to q and returns the
// modified url.Values.
func AddOptionalURLParams(q url.Values, opts []types.Option) url.Values {
	opt := &types.APIOptions{}
	for _, o := range opts {
		o(opt)
	}

	if opt.Timeout > 0 {
		q.Set("timeout", opt.Timeout.String())
	}

	if opt.LookbackDelta > 0 {
		q.Set("lookback_delta", opt.LookbackDelta.String())
	}

	if opt.Stats != "" {
		q.Set("stats", string(opt.Stats))
	}

	if opt.Limit > 0 {
		q.Set("limit", strconv.FormatUint(opt.Limit, 10))
	}

	return q
}

// FormatTime formats a time.Time as a decimal seconds string suitable for
// Prometheus API query parameters.
func FormatTime(t time.Time) string {
	return strconv.FormatFloat(float64(t.Unix())+float64(t.Nanosecond())/1e9, 'f', -1, 64)
}
