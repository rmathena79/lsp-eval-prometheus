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
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	apiPrefix = "/api/v1"

	Alerts          = apiPrefix + "/alerts"
	AlertManagers   = apiPrefix + "/alertmanagers"
	Query           = apiPrefix + "/query"
	QueryRange      = apiPrefix + "/query_range"
	QueryExemplars  = apiPrefix + "/query_exemplars"
	Labels          = apiPrefix + "/labels"
	LabelValues     = apiPrefix + "/label/:name/values"
	Series          = apiPrefix + "/series"
	Targets         = apiPrefix + "/targets"
	TargetsMetadata = apiPrefix + "/targets/metadata"
	Metadata        = apiPrefix + "/metadata"
	Rules           = apiPrefix + "/rules"
	Snapshot        = apiPrefix + "/admin/tsdb/snapshot"
	DeleteSeries    = apiPrefix + "/admin/tsdb/delete_series"
	CleanTombstones = apiPrefix + "/admin/tsdb/clean_tombstones"
	Config          = apiPrefix + "/status/config"
	Flags           = apiPrefix + "/status/flags"
	Buildinfo       = apiPrefix + "/status/buildinfo"
	Runtimeinfo     = apiPrefix + "/status/runtimeinfo"
	TSDB            = apiPrefix + "/status/tsdb"
	TSDBBlocks      = apiPrefix + "/status/tsdb/blocks"
	WalReplay       = apiPrefix + "/status/walreplay"
	FormatQuery     = apiPrefix + "/format_query"
)

type URLBuilder interface {
	URL(ep string, args map[string]string) *url.URL
}

func URL(client URLBuilder, ep string, args map[string]string, query url.Values) *url.URL {
	u := client.URL(ep, args)
	if query != nil {
		u.RawQuery = query.Encode()
	}
	return u
}

func NewGetRequest(client URLBuilder, ep string, args map[string]string, query url.Values) (*http.Request, error) {
	u := URL(client, ep, args, query)
	return http.NewRequest(http.MethodGet, u.String(), nil)
}

func NewPostRequest(client URLBuilder, ep string, args map[string]string, query url.Values, body io.Reader) (*http.Request, error) {
	u := URL(client, ep, args, query)
	return http.NewRequest(http.MethodPost, u.String(), body)
}

func FormatTime(t time.Time) string {
	return strconv.FormatFloat(float64(t.Unix())+float64(t.Nanosecond())/1e9, 'f', -1, 64)
}
