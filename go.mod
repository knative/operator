module knative.dev/operator

go 1.13

require (
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-logr/zapr v0.1.1
	github.com/go-openapi/spec v0.19.5
	github.com/go-openapi/swag v0.19.6 // indirect
	github.com/google/go-cmp v0.4.0
	github.com/google/licenseclassifier v0.0.0-20200402202327-879cb1424de0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.12.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/manifestival/client-go-client v0.2.0
	github.com/manifestival/manifestival v0.5.0
	github.com/operator-framework/operator-sdk v0.15.1 // indirect
	github.com/pkg/errors v0.9.1
	go.opencensus.io v0.22.3
	go.uber.org/atomic v1.5.1 // indirect
	go.uber.org/multierr v1.4.0 // indirect
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20200117160349-530e935923ad // indirect
	gopkg.in/yaml.v2 v2.2.7
	istio.io/api v0.0.0-20191115173247-e1a1952e5b81
	istio.io/client-go v0.0.0-20191120150049-26c62a04cdbc
	istio.io/gogo-genproto v0.0.0-20191029161641-f7d19ec0141d // indirect
	k8s.io/api v0.16.4
	k8s.io/apimachinery v0.16.5-beta.1
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.18.0
	k8s.io/kube-openapi v0.0.0-20190918143330-0270cf2f1c1d
	k8s.io/utils v0.0.0-20200122174043-1e243dd1a584 // indirect
	knative.dev/caching v0.0.0-20200122154023-853d6022845c
	knative.dev/eventing v0.14.1-0.20200424141350-b0da8720bf16
	knative.dev/pkg v0.0.0-20200427161250-c8577ac75766
	knative.dev/test-infra v0.0.0-20200427164251-3205a40d6171
	sigs.k8s.io/yaml v1.1.0
)

replace (
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
	k8s.io/api => k8s.io/api v0.16.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.16.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.4
	k8s.io/apiserver => k8s.io/apiserver v0.16.4
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.16.4
	k8s.io/client-go => k8s.io/client-go v0.16.4
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.16.4
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.16.4
	k8s.io/code-generator => k8s.io/code-generator v0.16.4
	k8s.io/component-base => k8s.io/component-base v0.16.4
	k8s.io/cri-api => k8s.io/cri-api v0.16.4
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.16.4
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.16.4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.16.4
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.16.4
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.16.4
	k8s.io/kubectl => k8s.io/kubectl v0.16.4
	k8s.io/kubelet => k8s.io/kubelet v0.16.4
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.16.4
	k8s.io/metrics => k8s.io/metrics v0.16.4
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.16.4
	knative.dev/pkg => github.com/chizhg/pkg v0.0.0-20200420011907-9117cd5cf224
)
