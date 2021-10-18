module knative.dev/operator

go 1.16

require (
	github.com/go-logr/zapr v0.4.0
	github.com/google/go-cmp v0.5.6
	github.com/google/go-github/v33 v33.0.0
	github.com/manifestival/client-go-client v0.5.0
	github.com/manifestival/manifestival v0.7.0
	go.uber.org/zap v1.19.1
	gocloud.dev v0.22.0
	golang.org/x/mod v0.4.2
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f
	k8s.io/api v0.21.4
	k8s.io/apimachinery v0.21.4
	k8s.io/client-go v0.21.4
	k8s.io/code-generator v0.21.4
	knative.dev/caching v0.0.0-20211015211723-5bfedd5e159d
	knative.dev/eventing v0.26.1-0.20211014072442-a6a819dc71cf
	knative.dev/hack v0.0.0-20211015200324-86876688e735
	knative.dev/pkg v0.0.0-20211015194524-a5bb75923981
	knative.dev/serving v0.26.1-0.20211016013324-e5d8560f950c
	sigs.k8s.io/yaml v1.3.0
)
