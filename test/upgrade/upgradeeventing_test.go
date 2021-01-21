// +build upgradeeventing

/*
Copyright 2020 The Knative Authors

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

package upgrade

import (
	"testing"

	"go.uber.org/zap"
	eventingupgrade "knative.dev/eventing/test/upgrade"
	"knative.dev/operator/test/upgrade/installation"
	pkgupgrade "knative.dev/pkg/test/upgrade"
)

func TestEventingUpgrades(t *testing.T) {
	suite := pkgupgrade.Suite{
		Tests: pkgupgrade.Tests{
			PreUpgrade: []pkgupgrade.Operation{
				EventingPreUpgradeTests(),
				eventingupgrade.PreUpgradeTest(),
			},
			PostUpgrade: []pkgupgrade.Operation{
				EventingPostUpgradeTests(),
				eventingupgrade.PostUpgradeTest(),
			},
			PostDowngrade: []pkgupgrade.Operation{
				EventingPostDowngradeTests(),
				eventingupgrade.PostDowngradeTest(),
			},
			Continual: []pkgupgrade.BackgroundOperation{
				eventingupgrade.ContinualTest(),
			},
		},
		Installations: pkgupgrade.Installations{
			Base: []pkgupgrade.Operation{},
			UpgradeWith: []pkgupgrade.Operation{
				installation.LatestRelease(),
			},
			DowngradeWith: []pkgupgrade.Operation{
				installation.PreviousRelease(),
			},
		},
	}
	c := newEventingUpgradeConfig(t)
	suite.Execute(c)
}

func newEventingUpgradeConfig(t *testing.T) pkgupgrade.Configuration {
	log, err := zap.NewDevelopment()
	if err != nil {
		t.Fatal(err)
	}
	return pkgupgrade.Configuration{T: t, Log: log}
}
