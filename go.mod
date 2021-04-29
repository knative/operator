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
	knative.dev/caching v0.0.0-20210428140754-1fab472d2faf
	knative.dev/eventing v0.22.1-0.20210428222054-c2d6649ea36d
	knative.dev/hack v0.0.0-20210428122153-93ad9129c268
	knative.dev/pkg v0.0.0-20210428141353-878c85083565
	knative.dev/serving v0.22.1-0.20210428225554-f60aa3b67816
	sigs.k8s.io/yaml v1.2.0
)
