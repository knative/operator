// +build upgradeserving

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
	"knative.dev/operator/test/upgrade/installation"
	pkgupgrade "knative.dev/pkg/test/upgrade"
	servingupgrade "knative.dev/serving/test/upgrade"
)

func TestServingUpgrades(t *testing.T) {
	suite := pkgupgrade.Suite{
		Tests: pkgupgrade.Tests{
			PreUpgrade: append([]pkgupgrade.Operation{
				ServingCRPreUpgradeTests(),
			}, servingupgrade.ServingPreUpgradeTests()...),
			PostUpgrade: append([]pkgupgrade.Operation{
				ServingCRPostUpgradeTests(),
			}, servingupgrade.ServingPostUpgradeTests()...),
			PostDowngrade: append([]pkgupgrade.Operation{
				ServingCRPostDowngradeTests(),
			}, servingupgrade.ServingPostDowngradeTests()...),
			Continual: servingupgrade.ContinualTests(),
		},
		Installations: pkgupgrade.Installations{
			Base: []pkgupgrade.Operation{
				installation.Base(),
				installation.ServingTestSetup(),
			},
			UpgradeWith: []pkgupgrade.Operation{
				installation.LatestRelease(),
			},
			DowngradeWith: []pkgupgrade.Operation{
				installation.PreviousRelease(),
			},
		},
	}
	c := newServingUpgradeConfig(t)
	suite.Execute(c)
}

func newServingUpgradeConfig(t *testing.T) pkgupgrade.Configuration {
	log, err := zap.NewDevelopment()
	if err != nil {
		t.Fatal(err)
	}
	return pkgupgrade.Configuration{T: t, Log: log}
}
