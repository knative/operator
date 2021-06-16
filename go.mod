module knative.dev/operator

go 1.16

require (
	github.com/go-logr/zapr v0.4.0
	github.com/google/go-cmp v0.5.6
	github.com/google/go-github/v33 v33.0.0
	github.com/manifestival/client-go-client v0.5.0
	github.com/manifestival/manifestival v0.7.0
	go.uber.org/zap v1.17.0
	gocloud.dev v0.22.0
	golang.org/x/mod v0.4.2
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c
	k8s.io/api v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/client-go v0.20.7
	k8s.io/code-generator v0.20.7
	knative.dev/caching v0.0.0-20210615114821-24a834ebd6d6
	knative.dev/eventing v0.23.1-0.20210616120222-193809a3b9c4
	knative.dev/hack v0.0.0-20210614141220-66ab1a098940
	knative.dev/pkg v0.0.0-20210615143321-77ff8d962c73
	knative.dev/serving v0.23.1-0.20210616130529-fb382393eedd
	sigs.k8s.io/yaml v1.2.0
)
