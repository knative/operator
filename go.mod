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
	k8s.io/api v0.23.5
	k8s.io/apimachinery v0.23.5
	k8s.io/client-go v0.23.5
	k8s.io/code-generator v0.23.5
	knative.dev/caching v0.0.0-20220412163508-8b5c244b8182
	knative.dev/eventing v0.30.1-0.20220414105410-5bb50cb1333d
	knative.dev/hack v0.0.0-20220411131823-6ffd8417de7c
	knative.dev/pkg v0.0.0-20220412134708-e325df66cb51
	knative.dev/serving v0.30.1-0.20220414210010-f931c3a82617
	sigs.k8s.io/yaml v1.3.0
)
