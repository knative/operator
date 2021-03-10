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
	knative.dev/caching v0.0.0-20210309014425-f75d758f5941
	knative.dev/eventing v0.21.1-0.20210309225325-879407f613a0
	knative.dev/hack v0.0.0-20210309141825-9b73a256fd9a
	knative.dev/pkg v0.0.0-20210309024624-0f8d8de5949d
	knative.dev/serving v0.21.1-0.20210310020825-86cc36dd4f65
	sigs.k8s.io/yaml v1.2.0
)
