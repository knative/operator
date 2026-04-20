# Knative Operator

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/knative/operator)
[![Go Report Card](https://goreportcard.com/badge/knative/operator)](https://goreportcard.com/report/knative/operator)
[![Releases](https://img.shields.io/github/release-pre/knative/operator.svg?sort=semver)](https://github.com/knative/operator/releases)
[![LICENSE](https://img.shields.io/github/license/knative/operator.svg)](https://github.com/knative/operator/blob/main/LICENSE)
[![Slack Status](https://img.shields.io/badge/slack-join_chat-white.svg?logo=slack&style=social)](https://knative.slack.com)
[![codecov](https://codecov.io/gh/knative/operator/branch/main/graph/badge.svg)](https://codecov.io/gh/knative/operator)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/5913/badge)](https://bestpractices.coreinfrastructure.org/projects/5913)

The Knative Operator defines custom resources for the
[Knative](https://knative.dev/) components, including serving and eventing, enabling users to configure, install,
upgrade and maintain these components over their lifecycle through a simple API.

Details:

- [Installation](https://knative.dev/docs/install/operator/knative-with-operators/)
- [Serving Configuration](https://knative.dev/docs/install/operator/configuring-serving-cr/)
- [Eventing Configuration](https://knative.dev/docs/install/operator/configuring-eventing-cr/)
- [Upgrade](docs/upgrade.md)
- [Multi-cluster deployment](docs/multicluster.md)
- [Development](docs/development.md)
- [Multi-cluster E2E testing](docs/development/e2e-multicluster.md)
- [Release](docs/release.md)

For documentation on using Knative Operator, see the
[Knative operator section](https://knative.dev/docs/install/operator/knative-with-operators/) of the
[Knative documentation site](https://www.knative.dev/docs).

## Operator CLI flags

Operator-specific CLI flags (set on the operator Deployment via `args:`):

| Flag | Default | Description |
|------|---------|-------------|
| `--clusterprofile-provider-file` | `""` | Path to the JSON config file describing Cluster Inventory API access providers. Required when any CR sets `spec.clusterProfileRef`. See [docs/multicluster.md](docs/multicluster.md). |
| `--remote-deployments-poll-interval` | `10s` | Requeue interval used while polling spoke deployment readiness. Raise for large fleets (`30s` for 10-100 spokes, `60s` for >100). Values below `1s` fall back to the default with a WARNING log entry. |

If you are interested in contributing, see [CONTRIBUTING.md](./CONTRIBUTING.md)
and [DEVELOPMENT.md](./DEVELOPMENT.md). For a list of help wanted issues across 
Knative, take a look at [CLOTRIBUTOR](https://clotributor.dev/search?project=knative&page=1).
