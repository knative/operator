module knative.dev/operator

go 1.16

require (
	cloud.google.com/go/storage v1.18.2 // indirect
	github.com/go-logr/zapr v0.4.0
	github.com/google/go-cmp v0.5.6
	github.com/google/go-github/v33 v33.0.0
	github.com/manifestival/client-go-client v0.5.0
	github.com/manifestival/manifestival v0.7.0
	github.com/prometheus/common v0.31.1 // indirect
	go.uber.org/zap v1.19.1
	gocloud.dev v0.22.0
	golang.org/x/mod v0.4.2
	golang.org/x/net v0.0.0-20211015210444-4f30a5c0130f // indirect
	golang.org/x/oauth2 v0.0.0-20211005180243-6b3c2da341f1
	golang.org/x/tools v0.1.7 // indirect
	google.golang.org/grpc v1.41.0 // indirect
	k8s.io/api v0.21.4
	k8s.io/apimachinery v0.21.4
	k8s.io/client-go v0.21.4
	k8s.io/code-generator v0.21.4
	k8s.io/gengo v0.0.0-20210915205010-39e73c8a59cd // indirect
	knative.dev/caching v0.0.0-20210914230307-0184eb914a42
	knative.dev/eventing v0.26.1
	knative.dev/hack v0.0.0-20210806075220-815cd312d65c
	knative.dev/pkg v0.0.0-20210919202233-5ae482141474
	knative.dev/serving v0.26.0
	sigs.k8s.io/yaml v1.3.0
)
