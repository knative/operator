/*
Copyright 2022 The Knative Authors

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

package base

// AwssqsSourceConfiguration specifies whether to enable the awssqs source.
type AwssqsSourceConfiguration struct {
	Enabled bool `json:"enabled"`
}

// CephSourceConfiguration specifies whether to enable the ceph source.
type CephSourceConfiguration struct {
	Enabled bool `json:"enabled"`
}

// CouchdbSourceConfiguration specifies whether to enable the couchdb source.
type CouchdbSourceConfiguration struct {
	Enabled bool `json:"enabled"`
}

// GithubSourceConfiguration specifies whether to enable the github source.
type GithubSourceConfiguration struct {
	Enabled bool `json:"enabled"`
}

// GitlabSourceConfiguration specifies whether to enable the gitlab source.
type GitlabSourceConfiguration struct {
	Enabled bool `json:"enabled"`
}

// KafkaSourceConfiguration specifies whether to enable the kafka source.
type KafkaSourceConfiguration struct {
	Enabled bool `json:"enabled"`
}

// NatssSourceConfiguration specifies whether to enable the natss source.
type NatssSourceConfiguration struct {
	Enabled bool `json:"enabled"`
}

// PrometheusSourceConfiguration specifies whether to enable the prometheus source.
type PrometheusSourceConfiguration struct {
	Enabled bool `json:"enabled"`
}

// RabbitmqSourceConfiguration specifies whether to enable the rabbitmq source.
type RabbitmqSourceConfiguration struct {
	Enabled bool `json:"enabled"`
}

// RedisSourceConfiguration specifies whether to enable the redis source.
type RedisSourceConfiguration struct {
	Enabled bool `json:"enabled"`
}
