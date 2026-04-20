/*
Copyright 2025 The Knative Authors

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
	"flag"
	"time"
)

const defaultRemoteDeploymentsPollInterval = 10 * time.Second

var remoteDeploymentsPollIntervalFlag time.Duration

func init() {
	flag.DurationVar(&remoteDeploymentsPollIntervalFlag,
		"remote-deployments-poll-interval", defaultRemoteDeploymentsPollInterval,
		"Interval for polling remote cluster deployments readiness. "+
			"Larger values reduce reconcile traffic when managing many spoke clusters. "+
			"Values below 1s fall back to the default.")
}

// RemoteDeploymentsPollIntervalValue returns the configured poll interval, clamping values below 1s to the default.
func RemoteDeploymentsPollIntervalValue() time.Duration {
	if remoteDeploymentsPollIntervalFlag < time.Second {
		return defaultRemoteDeploymentsPollInterval
	}
	return remoteDeploymentsPollIntervalFlag
}
