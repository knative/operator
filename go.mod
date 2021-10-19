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
	golang.org/x/oauth2 v0.0.0-20211005180243-6b3c2da341f1
	k8s.io/api v0.21.4
	k8s.io/apimachinery v0.21.4
	k8s.io/client-go v0.21.4
	k8s.io/code-generator v0.21.4
	knative.dev/caching v0.0.0-20211018141327-e93b91e2131e
	knative.dev/eventing v0.26.1-0.20211019092333-7af98bbb4491
	knative.dev/hack v0.0.0-20211019034732-ced8ce706528
	knative.dev/pkg v0.0.0-20211019101734-caa95feec115
	knative.dev/serving v0.26.1-0.20211019063033-2446281d75da
	sigs.k8s.io/yaml v1.3.0
)
