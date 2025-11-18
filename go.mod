module knative.dev/operator

go 1.24.0

require (
	github.com/go-logr/zapr v1.3.0
	github.com/google/go-cmp v0.7.0
	github.com/google/go-github/v33 v33.0.0
	github.com/hashicorp/golang-lru v1.0.2
	github.com/manifestival/client-go-client v0.6.0
	github.com/manifestival/manifestival v0.7.2
	go.uber.org/zap v1.27.0
	gocloud.dev v0.22.0
	golang.org/x/mod v0.29.0
	golang.org/x/oauth2 v0.30.0
	google.golang.org/api v0.198.0
	istio.io/api v0.0.0-20231206023236-e7cadb36da57
	istio.io/client-go v1.18.7
	k8s.io/api v0.34.1
	k8s.io/apimachinery v0.34.1
	k8s.io/client-go v0.34.1
	k8s.io/code-generator v0.34.1
	knative.dev/caching v0.0.0-20251117013858-60da9753c6da
	knative.dev/eventing v0.47.1-0.20251117120313-e0badc09375a
	knative.dev/hack v0.0.0-20251117013754-67a5b048bc27
	knative.dev/pkg v0.0.0-20251117013452-7e6a0208dd36
	knative.dev/reconciler-test v0.0.0-20251117013655-1e6de0a87d9a
	knative.dev/serving v0.47.1-0.20251117205120-9efd94748850
	sigs.k8s.io/yaml v1.6.0
)

require (
	cloud.google.com/go v0.115.1 // indirect
	cloud.google.com/go/auth v0.9.4 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.4 // indirect
	cloud.google.com/go/compute/metadata v0.7.0 // indirect
	cloud.google.com/go/iam v1.2.0 // indirect
	cloud.google.com/go/storage v1.43.0 // indirect
	github.com/Azure/azure-pipeline-go v0.2.3 // indirect
	github.com/Azure/azure-storage-blob-go v0.13.0 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.30 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.24 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.2 // indirect
	github.com/Azure/go-autorest/tracing v0.6.1 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr v1.4.10 // indirect
	github.com/aws/aws-sdk-go v1.36.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/blendle/zapdriver v1.3.1 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cert-manager/cert-manager v1.16.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudevents/conformance v0.4.1 // indirect
	github.com/cloudevents/sdk-go/observability/opentelemetry/v2 v2.16.1 // indirect
	github.com/cloudevents/sdk-go/sql/v2 v2.0.0-20240712172937-3ce6b2f1f011 // indirect
	github.com/cloudevents/sdk-go/v2 v2.16.1 // indirect
	github.com/coreos/go-oidc/v3 v3.9.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emicklei/go-restful/v3 v3.12.2 // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/go-jose/go-jose/v3 v3.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/gnostic-models v0.7.0 // indirect
	github.com/google/go-containerregistry v0.20.3 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/s2a-go v0.1.8 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/google/wire v0.4.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.4 // indirect
	github.com/googleapis/gax-go/v2 v2.13.0 // indirect
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.2 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7 // indirect
	github.com/influxdata/tdigest v0.0.1 // indirect
	github.com/jmespath/go-jmespath v0.4.1-0.20220621161143-b0104c826a24 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mattn/go-ieproxy v0.0.1 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/pelletier/go-toml/v2 v2.0.5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/otlptranslator v1.0.0 // indirect
	github.com/prometheus/procfs v0.17.0 // indirect
	github.com/rickb777/date v1.13.0 // indirect
	github.com/rickb777/plural v1.2.1 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/rs/dnscache v0.0.0-20230804202142-fc85eb664529 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/tsenart/vegeta/v12 v12.13.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/wavesoftware/go-ensure v1.0.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.60.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.63.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/runtime v0.63.0 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.60.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.38.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/sdk v1.38.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	go.opentelemetry.io/proto/otlp v1.7.1 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/term v0.36.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	golang.org/x/tools v0.38.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	gomodules.xyz/jsonpatch/v2 v2.5.0 // indirect
	google.golang.org/genproto v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250825161204-c5933d9347a5 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250825161204-c5933d9347a5 // indirect
	google.golang.org/grpc v1.75.0 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.12.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.34.1 // indirect
	k8s.io/apiserver v0.34.1 // indirect
	k8s.io/gengo/v2 v2.0.0-20250604051438-85fd79dbfd9f // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-openapi v0.0.0-20250710124328-f3f2b991d03b // indirect
	k8s.io/utils v0.0.0-20250604170112-4c0f3b243397 // indirect
	knative.dev/networking v0.0.0-20251117014740-e8506646d57f // indirect
	sigs.k8s.io/controller-runtime v0.19.0 // indirect
	sigs.k8s.io/gateway-api v1.1.0 // indirect
	sigs.k8s.io/json v0.0.0-20241014173422-cfa47c3a1cc8 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.3.0 // indirect
)
