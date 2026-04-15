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

import internaltypes "github.com/prometheus/client_golang/api/prometheus/v1/internal/types"

// AlertState models the state of an alert.
type AlertState = internaltypes.AlertState

// ErrorType models the different API error types.
type ErrorType = internaltypes.ErrorType

// HealthStatus models the health status of a scrape target.
type HealthStatus = internaltypes.HealthStatus

// RuleType models the type of a rule.
type RuleType = internaltypes.RuleType

// RuleHealth models the health status of a rule.
type RuleHealth = internaltypes.RuleHealth

// MetricType models the type of a metric.
type MetricType = internaltypes.MetricType

const (
	AlertStateFiring   AlertState = internaltypes.AlertStateFiring
	AlertStateInactive AlertState = internaltypes.AlertStateInactive
	AlertStatePending  AlertState = internaltypes.AlertStatePending

	ErrBadData     ErrorType = internaltypes.ErrBadData
	ErrTimeout     ErrorType = internaltypes.ErrTimeout
	ErrCanceled    ErrorType = internaltypes.ErrCanceled
	ErrExec        ErrorType = internaltypes.ErrExec
	ErrBadResponse ErrorType = internaltypes.ErrBadResponse
	ErrServer      ErrorType = internaltypes.ErrServer
	ErrClient      ErrorType = internaltypes.ErrClient

	HealthGood    HealthStatus = internaltypes.HealthGood
	HealthUnknown HealthStatus = internaltypes.HealthUnknown
	HealthBad     HealthStatus = internaltypes.HealthBad

	RuleTypeRecording RuleType = internaltypes.RuleTypeRecording
	RuleTypeAlerting  RuleType = internaltypes.RuleTypeAlerting

	RuleHealthGood    = internaltypes.RuleHealthGood
	RuleHealthUnknown = internaltypes.RuleHealthUnknown
	RuleHealthBad     = internaltypes.RuleHealthBad

	MetricTypeCounter        MetricType = internaltypes.MetricTypeCounter
	MetricTypeGauge          MetricType = internaltypes.MetricTypeGauge
	MetricTypeHistogram      MetricType = internaltypes.MetricTypeHistogram
	MetricTypeGaugeHistogram MetricType = internaltypes.MetricTypeGaugeHistogram
	MetricTypeSummary        MetricType = internaltypes.MetricTypeSummary
	MetricTypeInfo           MetricType = internaltypes.MetricTypeInfo
	MetricTypeStateset       MetricType = internaltypes.MetricTypeStateset
	MetricTypeUnknown        MetricType = internaltypes.MetricTypeUnknown
)

// Error is an error returned by the API.
type Error = internaltypes.Error

// Range represents a sliced time range.
type Range = internaltypes.Range

// AlertsResult contains the result from querying the alerts endpoint.
type AlertsResult = internaltypes.AlertsResult

// AlertManagersResult contains the result from querying the alertmanagers endpoint.
type AlertManagersResult = internaltypes.AlertManagersResult

// AlertManager models a configured Alert Manager.
type AlertManager = internaltypes.AlertManager

// ConfigResult contains the result from querying the config endpoint.
type ConfigResult = internaltypes.ConfigResult

// FlagsResult contains the result from querying the flag endpoint.
type FlagsResult = internaltypes.FlagsResult

// BuildinfoResult contains the results from querying the buildinfo endpoint.
type BuildinfoResult = internaltypes.BuildinfoResult

// RuntimeinfoResult contains the result from querying the runtimeinfo endpoint.
type RuntimeinfoResult = internaltypes.RuntimeinfoResult

// SnapshotResult contains the result from querying the snapshot endpoint.
type SnapshotResult = internaltypes.SnapshotResult

// RulesResult contains the result from querying the rules endpoint.
type RulesResult = internaltypes.RulesResult

// RuleGroup models a rule group that contains a set of recording and alerting rules.
type RuleGroup = internaltypes.RuleGroup

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
type Rules = internaltypes.Rules

// AlertingRule models a alerting rule.
type AlertingRule = internaltypes.AlertingRule

// RecordingRule models a recording rule.
type RecordingRule = internaltypes.RecordingRule

// Alert models an active alert.
type Alert = internaltypes.Alert

// TargetsResult contains the result from querying the targets endpoint.
type TargetsResult = internaltypes.TargetsResult

// ActiveTarget models an active Prometheus scrape target.
type ActiveTarget = internaltypes.ActiveTarget

// DroppedTarget models a dropped Prometheus scrape target.
type DroppedTarget = internaltypes.DroppedTarget

// MetricMetadata models the metadata of a metric with its scrape target and name.
type MetricMetadata = internaltypes.MetricMetadata

// Metadata models the metadata of a metric.
type Metadata = internaltypes.Metadata

// TSDBResult contains the result from querying the tsdb endpoint.
type TSDBResult = internaltypes.TSDBResult

// TSDBHeadStats contains TSDB stats.
type TSDBHeadStats = internaltypes.TSDBHeadStats

// TSDBBlocksResult contains the results from querying the tsdb blocks endpoint.
type TSDBBlocksResult = internaltypes.TSDBBlocksResult

// TSDBBlocksData contains the metadata for the tsdb blocks.
type TSDBBlocksData = internaltypes.TSDBBlocksData

// TSDBBlocksBlockMetadata contains the metadata for a single tsdb block.
type TSDBBlocksBlockMetadata = internaltypes.TSDBBlocksBlockMetadata

// TSDBBlocksStats contains block stats for a single tsdb block.
type TSDBBlocksStats = internaltypes.TSDBBlocksStats

// TSDBBlocksCompaction contains block compaction details for a single block.
type TSDBBlocksCompaction = internaltypes.TSDBBlocksCompaction

// WalReplayStatus represents the wal replay status.
type WalReplayStatus = internaltypes.WalReplayStatus

// Stat models information about statistic value.
type Stat = internaltypes.Stat

// Exemplar is additional information associated with a time series.
type Exemplar = internaltypes.Exemplar

// ExemplarQueryResult contains exemplars for a matching time series.
type ExemplarQueryResult = internaltypes.ExemplarQueryResult
