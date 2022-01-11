module knative.dev/operator

go 1.16

require (
	github.com/aws/aws-sdk-go v1.37.1 // indirect
	github.com/emicklei/go-restful v2.15.0+incompatible // indirect
	github.com/go-logr/zapr v0.4.0
	github.com/google/go-cmp v0.5.6
	github.com/google/go-github/v33 v33.0.0
	github.com/manifestival/client-go-client v0.5.0
	github.com/manifestival/manifestival v0.7.0
	go.uber.org/zap v1.19.1
	gocloud.dev v0.22.0
	golang.org/x/mod v0.5.1
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
	gonum.org/v1/gonum v0.0.0-20190331200053-3d26580ed485 // indirect
	k8s.io/api v0.22.5
	k8s.io/apimachinery v0.22.5
	k8s.io/client-go v0.22.5
	k8s.io/code-generator v0.22.5
	knative.dev/caching v0.0.0-20220105210833-30ba0b6609c6
	knative.dev/eventing v0.28.1-0.20220111105413-b5603c0ad63d
	knative.dev/hack v0.0.0-20220110200259-f08cb0dcdee7
	knative.dev/pkg v0.0.0-20220105211333-96f18522d78d
	knative.dev/serving v0.28.1-0.20220107170125-03091748d279
	sigs.k8s.io/yaml v1.3.0
)
