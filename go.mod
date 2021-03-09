module knative.dev/operator

go 1.15

require (
	github.com/go-logr/zapr v0.4.0
	github.com/google/go-cmp v0.5.4
	github.com/google/go-github/v33 v33.0.0
	github.com/manifestival/client-go-client v0.5.0
	github.com/manifestival/manifestival v0.7.0
	go.uber.org/zap v1.16.0
	gocloud.dev v0.22.0
	golang.org/x/mod v0.4.1
	golang.org/x/oauth2 v0.0.0-20210126194326-f9ce19ea3013
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.19.7
	k8s.io/client-go v0.19.7
	k8s.io/code-generator v0.19.7
	knative.dev/caching v0.0.0-20210308141422-49fcc5f83ec4
	knative.dev/eventing v0.21.1-0.20210309001301-65e14cf14b90
	knative.dev/hack v0.0.0-20210305150220-f99a25560134
	knative.dev/pkg v0.0.0-20210308172621-185e333b69ce
	knative.dev/serving v0.21.1-0.20210308183723-d51a7cc3d101
	sigs.k8s.io/yaml v1.2.0
)
