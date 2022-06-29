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
	go.uber.org/zap v1.21.0
	gocloud.dev v0.22.0
	golang.org/x/mod v0.5.1
	golang.org/x/oauth2 v0.0.0-20220223155221-ee480838109b
	gonum.org/v1/gonum v0.0.0-20190331200053-3d26580ed485 // indirect
	istio.io/api v0.0.0-20220420164308-b6a03a9e477e
	istio.io/client-go v1.13.3
	k8s.io/api v0.23.8
	k8s.io/apimachinery v0.23.8
	k8s.io/client-go v0.23.8
	k8s.io/code-generator v0.23.8
	knative.dev/caching v0.0.0-20220629132325-4bcd2765fffd
	knative.dev/eventing v0.32.1-0.20220629063230-9c41f7594cae
	knative.dev/hack v0.0.0-20220629132331-33ce6afd6f93
	knative.dev/pkg v0.0.0-20220629132331-f4ae0aff5b9c
	knative.dev/serving v0.32.1-0.20220629120131-d9cb5f0193f9
	sigs.k8s.io/yaml v1.3.0
)
