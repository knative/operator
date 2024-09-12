//go:build upgradeeventing
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
	"slices"
	"testing"

	eventingupgrade "knative.dev/eventing/test/upgrade"
	"knative.dev/operator/test/upgrade/installation"
	pkgupgrade "knative.dev/pkg/test/upgrade"
	"knative.dev/reconciler-test/pkg/environment"
)

var global environment.GlobalEnvironment

func TestEventingUpgrades(t *testing.T) {
	g := eventingupgrade.FeatureGroupWithUpgradeTests{
		// A feature that will run the same test post-upgrade and post-downgrade.
		eventingupgrade.NewFeatureSmoke(eventingupgrade.InMemoryChannelFeature(global)),
		// A feature that will be created pre-upgrade and verified/removed post-upgrade.
		eventingupgrade.NewFeatureOnlyUpgrade(eventingupgrade.InMemoryChannelFeature(global)),
		// A feature that will be created pre-upgrade, verified post-upgrade, verified and removed post-downgrade.
		eventingupgrade.NewFeatureUpgradeDowngrade(eventingupgrade.InMemoryChannelFeature(global)),
		// A feature that will be created post-upgrade, verified and removed post-downgrade.
		eventingupgrade.NewFeatureOnlyDowngrade(eventingupgrade.InMemoryChannelFeature(global)),
	}

	suite := pkgupgrade.Suite{
		Tests: pkgupgrade.Tests{
			PreUpgrade: slices.Concat(
				[]pkgupgrade.Operation{
					EventingCRPreUpgradeTests(),
				},
				g.PreUpgradeTests(),
			),
			PostUpgrade: slices.Concat(
				[]pkgupgrade.Operation{
					EventingTimeoutForUpgrade(),
					EventingCRPostUpgradeTests(),
					eventingupgrade.CRDPostUpgradeTest(),
				},
				g.PostUpgradeTests(),
			),
			PostDowngrade: slices.Concat(
				[]pkgupgrade.Operation{
					EventingCRPostDowngradeTests(),
				},
				g.PostDowngradeTests(),
			),
			Continual: []pkgupgrade.BackgroundOperation{
				eventingupgrade.ContinualTest(),
			},
		},
		Installations: pkgupgrade.Installations{
			Base: []pkgupgrade.Operation{
				installation.Base(),
			},
			UpgradeWith: []pkgupgrade.Operation{
				installation.LatestRelease(),
			},
			DowngradeWith: []pkgupgrade.Operation{
				installation.PreviousRelease(),
			},
		},
	}
	suite.Execute(pkgupgrade.Configuration{T: t})
}

func TestMain(m *testing.M) {
	eventingupgrade.RunMainTest(m)
}
