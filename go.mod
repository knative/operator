module knative.dev/operator

go 1.16

require (
	github.com/go-logr/zapr v0.4.0
	github.com/google/go-cmp v0.5.6
	github.com/google/go-github/v33 v33.0.0
	github.com/manifestival/client-go-client v0.5.0
	github.com/manifestival/manifestival v0.7.0
	go.uber.org/zap v1.18.1
	gocloud.dev v0.22.0
	golang.org/x/mod v0.4.2
	golang.org/x/oauth2 v0.0.0-20210628180205-a41e5a781914
	k8s.io/api v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/client-go v0.20.7
	k8s.io/code-generator v0.20.7
	knative.dev/caching v0.0.0-20210723125444-b6b4e59cc43e
	knative.dev/eventing v0.24.1-0.20210722165903-7251c1b9508f
	knative.dev/hack v0.0.0-20210622141627-e28525d8d260
	knative.dev/pkg v0.0.0-20210722223844-842df75f5c02
	knative.dev/serving v0.24.1-0.20210723122544-644798277d9f
	sigs.k8s.io/yaml v1.2.0
)
