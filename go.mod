module knative.dev/operator

go 1.15

require (
	github.com/go-logr/zapr v0.4.0
	github.com/google/go-cmp v0.5.5
	github.com/google/go-github/v33 v33.0.0
	github.com/manifestival/client-go-client v0.5.0
	github.com/manifestival/manifestival v0.7.0
	go.uber.org/zap v1.16.0
	gocloud.dev v0.22.0
	golang.org/x/mod v0.4.1
	golang.org/x/oauth2 v0.0.0-20210413134643-5e61552d6c78
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.19.7
	k8s.io/client-go v0.19.7
	k8s.io/code-generator v0.19.7
	knative.dev/caching v0.0.0-20210426131942-0643a27080f6
	knative.dev/eventing v0.22.1-0.20210426130639-83a1713973d5
	knative.dev/hack v0.0.0-20210426064739-88c69cd1eca7
	knative.dev/pkg v0.0.0-20210426101439-2a0fc657a712
	knative.dev/serving v0.22.1-0.20210426060740-792c8d233f49
	sigs.k8s.io/yaml v1.2.0
)
