module knative.dev/operator

go 1.15

require (
	github.com/go-logr/zapr v0.2.0
	github.com/google/go-cmp v0.5.4
	github.com/google/go-github/v32 v32.1.0
	github.com/manifestival/client-go-client v0.4.0
	github.com/manifestival/manifestival v0.6.1
	go.uber.org/zap v1.16.0
	golang.org/x/mod v0.4.1
	golang.org/x/oauth2 v0.0.0-20210126194326-f9ce19ea3013
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.19.7
	k8s.io/client-go v0.19.7
	k8s.io/code-generator v0.19.7
	knative.dev/caching v0.0.0-20210209013428-8ae1528470a4
	knative.dev/eventing v0.20.1-0.20210209233432-7ce82834b39d
	knative.dev/hack v0.0.0-20210203173706-8368e1f6eacf
	knative.dev/pkg v0.0.0-20210208175252-a02dcff9ee26
	knative.dev/serving v0.20.1-0.20210210103320-ebc658424a0d
	sigs.k8s.io/yaml v1.2.0
)
