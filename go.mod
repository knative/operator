module knative.dev/operator

go 1.14

require (
	github.com/emicklei/go-restful v2.11.1+incompatible // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-logr/zapr v0.1.1
	github.com/google/go-cmp v0.4.0
	github.com/grpc-ecosystem/grpc-gateway v1.12.2 // indirect
	github.com/manifestival/client-go-client v0.2.3-0.20200702141517-e255fbf14f6f
	github.com/manifestival/manifestival v0.5.1-0.20200702141132-93669fa9179b
	github.com/pkg/errors v0.9.1
	go.uber.org/zap v1.14.1
	golang.org/x/mod v0.3.0
	gonum.org/v1/gonum v0.0.0-20190710053202-4340aa3071a0 // indirect
	gopkg.in/yaml.v2 v2.2.8
	istio.io/api v0.0.0-20191115173247-e1a1952e5b81
	istio.io/client-go v0.0.0-20191120150049-26c62a04cdbc
	istio.io/gogo-genproto v0.0.0-20191029161641-f7d19ec0141d // indirect
	k8s.io/api v0.17.6
	k8s.io/apimachinery v0.17.6
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/code-generator v0.18.0
	knative.dev/caching v0.0.0-20200122154023-853d6022845c
	knative.dev/eventing v0.14.1-0.20200428210242-f355830c4d70
	knative.dev/pkg v0.0.0-20200625173728-dfb81cf04a7c
	knative.dev/test-infra v0.0.0-20200519161858-554a95a37986
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309

	k8s.io/api => k8s.io/api v0.17.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.6
	k8s.io/client-go => k8s.io/client-go v0.17.6
	k8s.io/code-generator => k8s.io/code-generator v0.17.6
)
