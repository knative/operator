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
	knative.dev/caching v0.0.0-20220704131745-bd32ea76811a
	knative.dev/eventing v0.32.1-0.20220704055944-a63ea4e0e4d0
	knative.dev/hack v0.0.0-20220701014203-65c463ac8c98
	knative.dev/pkg v0.0.0-20220701013933-97eb1507655e
	knative.dev/serving v0.32.1-0.20220704172911-6ceb219d443d
	sigs.k8s.io/yaml v1.3.0
)
