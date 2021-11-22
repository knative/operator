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
	golang.org/x/oauth2 v0.0.0-20211028175245-ba495a64dcb5
	k8s.io/api v0.21.4
	k8s.io/apimachinery v0.21.4
	k8s.io/client-go v0.21.4
	k8s.io/code-generator v0.21.4
	knative.dev/caching v0.0.0-20211122065213-d34fbbcce65b
	knative.dev/eventing v0.27.1-0.20211122062614-7c8aaa44e534
	knative.dev/hack v0.0.0-20211117134436-69a2295d54ce
	knative.dev/pkg v0.0.0-20211120133512-d016976f2567
	knative.dev/serving v0.27.1-0.20211122084314-5e96dc0e66fe
	sigs.k8s.io/yaml v1.3.0
)
