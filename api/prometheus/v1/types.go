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

package v1

import v1types "github.com/prometheus/client_golang/api/prometheus/v1/internal/types"

// AlertState models the state of an alert.
type AlertState = v1types.AlertState

// ErrorType models the different API error types.
type ErrorType = v1types.ErrorType

// HealthStatus models the health status of a scrape target.
type HealthStatus = v1types.HealthStatus

// RuleType models the type of a rule.
type RuleType = v1types.RuleType

// RuleHealth models the health status of a rule.
type RuleHealth = v1types.RuleHealth

// MetricType models the type of a metric.
type MetricType = v1types.MetricType

const (
	// Possible values for AlertState.
	AlertStateFiring   AlertState = v1types.AlertStateFiring
	AlertStateInactive AlertState = v1types.AlertStateInactive
	AlertStatePending  AlertState = v1types.AlertStatePending

	// Possible values for ErrorType.
	ErrBadData     ErrorType = v1types.ErrBadData
	ErrTimeout     ErrorType = v1types.ErrTimeout
	ErrCanceled    ErrorType = v1types.ErrCanceled
	ErrExec        ErrorType = v1types.ErrExec
	ErrBadResponse ErrorType = v1types.ErrBadResponse
	ErrServer      ErrorType = v1types.ErrServer
	ErrClient      ErrorType = v1types.ErrClient

	// Possible values for HealthStatus.
	HealthGood    HealthStatus = v1types.HealthGood
	HealthUnknown HealthStatus = v1types.HealthUnknown
	HealthBad     HealthStatus = v1types.HealthBad

	// Possible values for RuleType.
	RuleTypeRecording RuleType = v1types.RuleTypeRecording
	RuleTypeAlerting  RuleType = v1types.RuleTypeAlerting

	// Possible values for RuleHealth.
	RuleHealthGood    = v1types.RuleHealthGood
	RuleHealthUnknown = v1types.RuleHealthUnknown
	RuleHealthBad     = v1types.RuleHealthBad

	// Possible values for MetricType
	MetricTypeCounter        MetricType = v1types.MetricTypeCounter
	MetricTypeGauge          MetricType = v1types.MetricTypeGauge
	MetricTypeHistogram      MetricType = v1types.MetricTypeHistogram
	MetricTypeGaugeHistogram MetricType = v1types.MetricTypeGaugeHistogram
	MetricTypeSummary        MetricType = v1types.MetricTypeSummary
	MetricTypeInfo           MetricType = v1types.MetricTypeInfo
	MetricTypeStateset       MetricType = v1types.MetricTypeStateset
	MetricTypeUnknown        MetricType = v1types.MetricTypeUnknown
)

// Error is an error returned by the API.
type Error = v1types.Error

// Range represents a sliced time range.
type Range = v1types.Range

// AlertsResult contains the result from querying the alerts endpoint.
type AlertsResult = v1types.AlertsResult

// AlertManagersResult contains the result from querying the alertmanagers endpoint.
type AlertManagersResult = v1types.AlertManagersResult

// AlertManager models a configured Alert Manager.
type AlertManager = v1types.AlertManager

// ConfigResult contains the result from querying the config endpoint.
type ConfigResult = v1types.ConfigResult

// FlagsResult contains the result from querying the flag endpoint.
type FlagsResult = v1types.FlagsResult

// BuildinfoResult contains the results from querying the buildinfo endpoint.
type BuildinfoResult = v1types.BuildinfoResult

// RuntimeinfoResult contains the result from querying the runtimeinfo endpoint.
type RuntimeinfoResult = v1types.RuntimeinfoResult

// SnapshotResult contains the result from querying the snapshot endpoint.
type SnapshotResult = v1types.SnapshotResult

// RulesResult contains the result from querying the rules endpoint.
type RulesResult = v1types.RulesResult

// RuleGroup models a rule group that contains a set of recording and alerting rules.
type RuleGroup = v1types.RuleGroup

// Recording and alerting rules are stored in the same slice to preserve the order
// that rules are returned in by the API.
//
// Rule types can be determined using a type switch:
//
//	switch v := rule.(type) {
//	case RecordingRule:
//		fmt.Print("got a recording rule")
//	case AlertingRule:
//		fmt.Print("got a alerting rule")
//	default:
//		fmt.Printf("unknown rule type %s", v)
//	}
type Rules = v1types.Rules

// AlertingRule models a alerting rule.
type AlertingRule = v1types.AlertingRule

// RecordingRule models a recording rule.
type RecordingRule = v1types.RecordingRule

// Alert models an active alert.
type Alert = v1types.Alert

// TargetsResult contains the result from querying the targets endpoint.
type TargetsResult = v1types.TargetsResult

// ActiveTarget models an active Prometheus scrape target.
type ActiveTarget = v1types.ActiveTarget

// DroppedTarget models a dropped Prometheus scrape target.
type DroppedTarget = v1types.DroppedTarget

// MetricMetadata models the metadata of a metric with its scrape target and name.
type MetricMetadata = v1types.MetricMetadata

// Metadata models the metadata of a metric.
type Metadata = v1types.Metadata

// TSDBResult contains the result from querying the tsdb endpoint.
type TSDBResult = v1types.TSDBResult

// TSDBHeadStats contains TSDB stats
type TSDBHeadStats = v1types.TSDBHeadStats

// TSDBBlocksResult contains the results from querying the tsdb blocks endpoint.
type TSDBBlocksResult = v1types.TSDBBlocksResult

// TSDBBlocksData contains the metadata for the tsdb blocks.
type TSDBBlocksData = v1types.TSDBBlocksData

// TSDBBlocksBlockMetadata contains the metadata for a single tsdb block.
type TSDBBlocksBlockMetadata = v1types.TSDBBlocksBlockMetadata

// TSDBBlocksStats contains block stats for a single tsdb block.
type TSDBBlocksStats = v1types.TSDBBlocksStats

// TSDBBlocksCompaction contains block compaction details for a single block.
type TSDBBlocksCompaction = v1types.TSDBBlocksCompaction

// WalReplayStatus represents the wal replay status.
type WalReplayStatus = v1types.WalReplayStatus

// Stat models information about statistic value.
type Stat = v1types.Stat

// Exemplar is additional information associated with a time series.
type Exemplar = v1types.Exemplar

type ExemplarQueryResult = v1types.ExemplarQueryResult

// StatsValue is a type for `stats` query parameter.
type StatsValue = v1types.StatsValue

// AllStatsValue is the query parameter value to return all the query statistics.
const (
	AllStatsValue StatsValue = v1types.AllStatsValue
)

// Warnings is an array of non critical errors
type Warnings = v1types.Warnings
