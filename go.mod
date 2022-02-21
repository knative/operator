module knative.dev/operator

go 1.16

require (
	github.com/aws/aws-sdk-go v1.37.1 // indirect
	github.com/emicklei/go-restful v2.15.0+incompatible // indirect
	github.com/go-logr/zapr v1.2.2
	github.com/google/go-cmp v0.5.6
	github.com/google/go-github/v33 v33.0.0
	github.com/manifestival/client-go-client v0.5.0
	github.com/manifestival/manifestival v0.7.1
	go.uber.org/zap v1.19.1
	gocloud.dev v0.22.0
	golang.org/x/mod v0.5.1
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
	gonum.org/v1/gonum v0.0.0-20190331200053-3d26580ed485 // indirect
	k8s.io/api v0.22.5
	k8s.io/apimachinery v0.22.5
	k8s.io/client-go v0.22.5
	k8s.io/code-generator v0.22.5
	knative.dev/caching v0.0.0-20220217152914-057ba67508ef
	knative.dev/eventing v0.29.1-0.20220217054812-7a48f4269b6f
	knative.dev/hack v0.0.0-20220218190734-a8ef7b67feec
	knative.dev/pkg v0.0.0-20220217155112-d48172451966
	knative.dev/serving v0.29.1-0.20220217223834-b28062cdc4c7
	sigs.k8s.io/yaml v1.3.0
)
