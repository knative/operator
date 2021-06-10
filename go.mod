module knative.dev/operator

go 1.16

require (
	github.com/go-logr/zapr v0.4.0
	github.com/google/go-cmp v0.5.6
	github.com/google/go-github/v33 v33.0.0
	github.com/manifestival/client-go-client v0.5.0
	github.com/manifestival/manifestival v0.7.0
	go.uber.org/zap v1.17.0
	gocloud.dev v0.22.0
	golang.org/x/mod v0.4.2
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c
	k8s.io/api v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/client-go v0.20.7
	k8s.io/code-generator v0.20.7
	knative.dev/caching v0.0.0-20210610054943-d7b71418f741
	knative.dev/eventing v0.23.1-0.20210610143343-393e4119bf65
	knative.dev/hack v0.0.0-20210609124042-e35bcb8f21ec
	knative.dev/pkg v0.0.0-20210610171444-c96c7beb09df
	knative.dev/serving v0.23.1-0.20210610163543-6064fc9ee446
	sigs.k8s.io/yaml v1.2.0
)
