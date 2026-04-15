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

const Prefix = "/api/v1"

const (
	Alerts          = Prefix + "/alerts"
	AlertManagers   = Prefix + "/alertmanagers"
	Query           = Prefix + "/query"
	QueryRange      = Prefix + "/query_range"
	QueryExemplars  = Prefix + "/query_exemplars"
	Labels          = Prefix + "/labels"
	LabelValues     = Prefix + "/label/:name/values"
	Series          = Prefix + "/series"
	Targets         = Prefix + "/targets"
	TargetsMetadata = Prefix + "/targets/metadata"
	Metadata        = Prefix + "/metadata"
	Rules           = Prefix + "/rules"
	Snapshot        = Prefix + "/admin/tsdb/snapshot"
	DeleteSeries    = Prefix + "/admin/tsdb/delete_series"
	CleanTombstones = Prefix + "/admin/tsdb/clean_tombstones"
	Config          = Prefix + "/status/config"
	Flags           = Prefix + "/status/flags"
	Buildinfo       = Prefix + "/status/buildinfo"
	Runtimeinfo     = Prefix + "/status/runtimeinfo"
	TSDB            = Prefix + "/status/tsdb"
	TSDBBlocks      = Prefix + "/status/tsdb/blocks"
	WalReplay       = Prefix + "/status/walreplay"
	FormatQuery     = Prefix + "/format_query"
)

func NewRequest(method string, u *url.URL) (*http.Request, error) {
	return http.NewRequest(method, u.String(), nil)
}

func SetQuery(u *url.URL, q url.Values) {
	u.RawQuery = q.Encode()
}

func FormatTime(t time.Time) string {
	return strconv.FormatFloat(float64(t.Unix())+float64(t.Nanosecond())/1e9, 'f', -1, 64)
}
