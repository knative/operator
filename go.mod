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
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.19.7
	k8s.io/client-go v0.19.7
	k8s.io/code-generator v0.19.7
	knative.dev/caching v0.0.0-20210201195732-f8bed817dad1
	knative.dev/eventing v0.20.1-0.20210202082232-a4a9f4875544
	knative.dev/hack v0.0.0-20210120165453-8d623a0af457
	knative.dev/pkg v0.0.0-20210130001831-ca02ef752ac6
	knative.dev/serving v0.20.1-0.20210202104232-6313088d8452
	sigs.k8s.io/yaml v1.2.0
)
