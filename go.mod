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
	golang.org/x/oauth2 v0.0.0-20210126194326-f9ce19ea3013
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.19.7
	k8s.io/client-go v0.19.7
	k8s.io/code-generator v0.19.7
	knative.dev/caching v0.0.0-20210318161455-3aa75bbb7d3d
	knative.dev/eventing v0.21.1-0.20210319025353-a95567e25e68
	knative.dev/hack v0.0.0-20210317214554-58edbdc42966
	knative.dev/pkg v0.0.0-20210318052054-dfeeb1817679
	knative.dev/serving v0.21.1-0.20210319035153-08e4e0e0021a
	sigs.k8s.io/yaml v1.2.0
)
