module knative.dev/operator

go 1.15

require (
	github.com/emicklei/go-restful v2.11.1+incompatible // indirect
	github.com/go-logr/zapr v0.2.0
	github.com/google/go-cmp v0.5.4
	github.com/google/go-github/v32 v32.1.0
	github.com/manifestival/client-go-client v0.4.0
	github.com/manifestival/manifestival v0.6.1
	go.uber.org/zap v1.16.0
	golang.org/x/mod v0.3.0
	golang.org/x/oauth2 v0.0.0-20201208152858-08078c50e5b5
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.19.7
	k8s.io/client-go v0.19.7
	k8s.io/code-generator v0.19.7
	knative.dev/caching v0.0.0-20210125050654-45e8de7ff96e
	knative.dev/eventing v0.20.1-0.20210128132430-1725902f7e39
	knative.dev/hack v0.0.0-20210120165453-8d623a0af457
	knative.dev/pkg v0.0.0-20210127163530-0d31134d5f4e
	knative.dev/serving v0.20.1-0.20210128192731-ab176fac7370
	sigs.k8s.io/yaml v1.2.0
)
