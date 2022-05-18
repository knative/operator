module knative.dev/operator

go 1.16

require (
	cloud.google.com/go/iam v0.2.0 // indirect
	github.com/aws/aws-sdk-go v1.37.1 // indirect
	github.com/emicklei/go-restful v2.15.0+incompatible // indirect
	github.com/go-logr/zapr v1.2.2
	github.com/google/go-cmp v0.5.7
	github.com/google/go-github/v33 v33.0.0
	github.com/manifestival/client-go-client v0.5.0
	github.com/manifestival/manifestival v0.7.1
	go.uber.org/zap v1.19.1
	gocloud.dev v0.22.0
	golang.org/x/mod v0.5.1
	golang.org/x/oauth2 v0.0.0-20220223155221-ee480838109b
	gonum.org/v1/gonum v0.0.0-20190331200053-3d26580ed485 // indirect
	istio.io/api v0.0.0-20220420164308-b6a03a9e477e
	istio.io/client-go v1.13.3
	k8s.io/api v0.23.5
	k8s.io/apimachinery v0.23.5
	k8s.io/client-go v0.23.5
	k8s.io/code-generator v0.23.5
	knative.dev/caching v0.0.0-20220518013728-03ca91f08490
	knative.dev/eventing v0.31.1-0.20220518013728-94491694bf3c
	knative.dev/hack v0.0.0-20220518013938-b4d4d73a2995
	knative.dev/pkg v0.0.0-20220518013728-f907b31046d3
	knative.dev/serving v0.31.1-0.20220518013906-4e54509b0a2c
	sigs.k8s.io/yaml v1.3.0
)
