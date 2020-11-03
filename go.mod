module knative.dev/operator

go 1.14

require (
	github.com/emicklei/go-restful v2.11.1+incompatible // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32 // indirect
	github.com/go-logr/zapr v0.1.1
	github.com/google/go-cmp v0.5.2
	github.com/manifestival/client-go-client v0.4.0
	github.com/manifestival/manifestival v0.6.1
	go.uber.org/zap v1.16.0
	golang.org/x/mod v0.3.0
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/code-generator v0.18.8
	knative.dev/caching v0.0.0-20201029165635-7983fcd03d68
	knative.dev/eventing v0.18.1-0.20201102224004-f0648377057c
	knative.dev/hack v0.0.0-20201102193445-9349aeeb6701
	knative.dev/pkg v0.0.0-20201102194004-c6a8219d518d
	sigs.k8s.io/yaml v1.2.0
)

replace (
	k8s.io/api => k8s.io/api v0.18.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.8
	k8s.io/client-go => k8s.io/client-go v0.18.8
	k8s.io/code-generator => k8s.io/code-generator v0.18.8
)
