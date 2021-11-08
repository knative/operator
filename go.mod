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
	knative.dev/caching v0.0.0-20211108065001-0c35ec4809b1
	knative.dev/eventing v0.27.1-0.20211108073202-ee96d0b3b429
	knative.dev/hack v0.0.0-20211105231158-29f86c2653b5
	knative.dev/pkg v0.0.0-20211108064904-3cc697a3cb09
	knative.dev/serving v0.27.1-0.20211108122300-af13029f220b
	sigs.k8s.io/yaml v1.3.0
)
