package endpoints

import (
	"net/url"
	"strconv"
	"time"
)

const apiPrefix = "/api/v1"

const (
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

func AddMatchers(q url.Values, matches []string) {
	for _, m := range matches {
		q.Add("match[]", m)
	}
}

func SetOptionalTime(q url.Values, key string, t time.Time) {
	if !t.IsZero() {
		q.Set(key, FormatTime(t))
	}
}

func FormatTime(t time.Time) string {
	return strconv.FormatFloat(float64(t.Unix())+float64(t.Nanosecond())/1e9, 'f', -1, 64)
}
