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

package common

import (
	"testing"

	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"knative.dev/pkg/metrics/metricstest"
)

const (
	testReconcilerName             = "test_reconciler"
	testKnativeservingResourceName = "ns/name"
)

func TestNewStatsReporter(t *testing.T) {
	r, err := NewStatsReporter(testReconcilerName)
	if err != nil {
		t.Errorf("Failed to create reporter: %v", err)
	}

	m := tag.FromContext(r.(*reporter).ctx)
	v, ok := m.Value(reconcilerNameTagKey)
	if !ok {
		t.Fatalf("Expected tag %q", reconcilerNameTagKey)
	}
	if v != testReconcilerName {
		t.Fatalf("Expected %q for tag %q, got %q", testReconcilerName, reconcilerNameTagKey, v)
	}
}

func TestReportKnativeServingChange(t *testing.T) {
	r, _ := NewStatsReporter(testReconcilerName)
	wantTags := map[string]string{
		reconcilerNameTagKey.Name():             testReconcilerName,
		knativeservingResourceNameTagKey.Name(): testKnativeservingResourceName,
		changeTagKey.Name():                     "creation",
	}
	countWas := int64(0)
	if d, err := view.RetrieveData(knativeservingChangeCountName); err == nil && len(d) == 1 {
		countWas = d[0].Data.(*view.CountData).Value
	}

	if err := r.ReportKnativeservingChange(testKnativeservingResourceName, "creation"); err != nil {
		t.Error(err)
	}

	metricstest.CheckCountData(t, knativeservingChangeCountName, wantTags, countWas+1)
}
