/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package reconciler

import (
	"context"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"knative.dev/pkg/metrics"
)

const (
	knativeservingChangeCountName = "knativeserving_change_count"
)

var (
	knativeservingChangeCountStat = stats.Int64(
		knativeservingChangeCountName,
		"Number of changes to the KnativeServing Custom Resource where a change can be creation, edit, or deletion",
		stats.UnitDimensionless)
	// Create the tag keys that will be used to add tags to our measurements.
	// Tag keys must conform to the restrictions described in
	// go.opencensus.io/tag/validate.go. Currently those restrictions are:
	// - length between 1 and 255 inclusive
	// - characters are printable US-ASCII
	reconcilerNameTagKey             = tag.MustNewKey("reconciler_name")
	knativeservingResourceNameTagKey = tag.MustNewKey("knativeserving_resource_name")
	changeTagKey                     = tag.MustNewKey("change")
)

func init() {
	// Create views to see our measurements. This can return an error if
	// a previously-registered view has the same name with a different value.
	// View name defaults to the measure name if unspecified.
	views := []*view.View{{
		Description: knativeservingChangeCountStat.Description(),
		Measure:     knativeservingChangeCountStat,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{reconcilerNameTagKey, knativeservingResourceNameTagKey, changeTagKey},
	}}

	if err := view.Register(views...); err != nil {
		panic(err)
	}
}

// StatsReporter defines the interface for sending metrics.
type StatsReporter interface {
	// ReportKnativeServingChange reports the count of KnativeServing changes.
	ReportKnativeservingChange(knativeservingResourceName, change string) error
}

// reporter holds cached metric objects to report metrics.
type reporter struct {
	reconcilerName string
	ctx            context.Context
}

// NewStatsReporter creates a reporter that collects and reports metrics.
func NewStatsReporter(reconcilerName string) (StatsReporter, error) {
	// Reconciler tag is static. Create a context containing that and cache it.
	ctx, err := tag.New(
		context.Background(),
		tag.Insert(reconcilerNameTagKey, reconcilerName))
	if err != nil {
		return nil, err
	}
	return &reporter{reconcilerName: reconcilerName, ctx: ctx}, nil
}

// ReportKnativeServingChange reports the number of changes to the KnativeServing
// Custom Resource where a change can be creation, edit, or deletion.
func (r *reporter) ReportKnativeservingChange(knativeservingResourceName, change string) error {
	ctx, err := tag.New(
		r.ctx,
		tag.Insert(knativeservingResourceNameTagKey, knativeservingResourceName),
		tag.Insert(changeTagKey, change))
	if err != nil {
		return err
	}

	metrics.Record(ctx, knativeservingChangeCountStat.M(1))
	return nil
}
