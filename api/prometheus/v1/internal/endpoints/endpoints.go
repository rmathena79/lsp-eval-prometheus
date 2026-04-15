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

// Package endpoints defines the HTTP API path constants for the Prometheus
// HTTP API v1. All paths are absolute (prefixed with /api/v1).
package endpoints

// APIPrefix is the common path prefix for all v1 API endpoints.
const APIPrefix = "/api/v1"

// Endpoint path constants for the Prometheus HTTP API v1.
const (
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
