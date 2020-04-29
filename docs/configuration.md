# Configuration

This document describes the `spec` sub-fields of the
[KnativeServing](../config/300-serving.yaml) and
[KnativeEventing](../config/300-eventing.yaml) custom resources. A
Knative installation is configured using these sub-fields.

The `kubectl explain` command shows short descriptions of the custom
resource fields installed on your cluster:

```
kubectl explain KnativeServing.spec
kubectl explain KnativeEventing.spec
```

If the output of those commands differs from this doc, you may need to
[upgrade](installation.md#upgrades) your operator.

These are the configurable fields in each resource:

* **KnativeServing**
  * `spec`
    * [config](#specconfig)
    * [registry](#specregistry)
      * [default](#specregistrydefault)
      * [override](#specregistryoverride)
      * [imagePullSecrets](#specregistryimagepullsecrets)
    * [controller-custom-certs](#speccontroller-custom-certs)
    * [knative-ingress-gateway](#specknative-ingress-gateway)
    * [cluster-local-gateway](#speccluster-local-gateway)
    * [high-availability](#spechigh-availability)
    * [resources](#specresources)
* **KnativeEventing**
  * `spec`
    * [config](#specconfig)
    * [registry](#specregistry)
      * [default](#specregistrydefault)
      * [override](#specregistryoverride)
      * [imagePullSecrets](#specregistryimagepullsecrets)
    * [resources](#specresources)
    * [defaultBrokerClass](#specdefaultbrokerclass)


## spec.config

This is a "map of maps". The top-level keys correspond to the names of
the Knative `ConfigMaps`, and the operator will ensure their values
replace the entries in the `data` field of the actual `ConfigMap`.
This provides a central place to manage all your Knative
configuration, without having to update multiple `ConfigMaps`.

If the name of the `ConfigMap` you wish to override begins with
`config-` (the Knative convention), that prefix may be omitted in the
top-level key name.

An example should help clarify. The following spec will cause the
operator to update the `config-defaults`, `config-observability` and
`config-autoscaler` `ConfigMaps` in the `knative` namespace with the
values you see below:

```
metadata:
  namespace: knative
spec:
  config:
    defaults:
      revision-timeout-seconds: "300"  # 5 minutes
    observability:
      metrics.backend-destination: prometheus
    autoscaler:
      stable-window: "60s"
      container-concurrency-target-default: '100'
```


## spec.registry

This field provides the ability to replace the images in the knative
deployment container specs via three optional sub-fields: `override`,
`default`, and `imagePullSecrets`.


### spec.registry.override

The optional `override` field is a mapping of deployment container
names to docker image names.

If the container names are not unique across all of your deployments,
you can prefix the container name with the deployment name and a
slash, e.g. `deployment/container`.

Because some container specs map environment variables to image names,
those are permitted as keys in the `override` map as well.

The following example overrides only the `autoscaler` container,
the `controller` container in the `broker` deployment spec, and the
value of the `DISPATCHER_IMAGE` env var.

```
spec:
  registry:
    override:
      autoscaler: docker.io/my-org/autoscaler:v0.13.0
      broker/controller: docker.io/my-org/broker-controller:v0.13.0
      DISPATCHER_IMAGE: docker.io/my-org/dispatcher:v0.13.0
```


### spec.registry.default

The `default` field enables you to override _all_ the knative
container images with minimal configuration. Its value is essentially
a pattern, matching all (or most) of the image names, with the
placeholder `${NAME}` representing the container's name. For example,

```
spec:
  registry:
    default: docker.io/my-org/knative-${NAME}:v1.0.0
```

The operator will replace _all_ image names with this value, after
replacing `${NAME}` with the container's actual name. 

Of course, for any images that don't match the pattern, you'll need to
provide entries in the [`override` field](#specregistryoverride). 

For example, if the name of the `autoscaler-hpa` image is different
than all the others, for whatever reason, you might have this:

```
spec:
  registry:
    default: docker.io/my-org/knative-serving-${NAME}:v1.0.0
    override:
      autoscaler-hpa: docker.io/my-org/ks-hpa:v1.0.1
```


### spec.registry.imagePullSecrets

This field is a list of `Secret` names required when pulling your
container images. You must create the secrets in the same namespace as
the knative resources.

- [From existing docker credentials](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#registry-secret-existing-credentials)
- [From command line for docker credentials](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#create-a-secret-by-providing-credentials-on-the-command-line)
- [Create your own secret](https://kubernetes.io/docs/concepts/configuration/secret/#creating-your-own-secrets)

Once created, simply refer to them in your spec:

```
spec:
  registry:
    default: docker.io/my-org/knative-${NAME}:v1.0.0
    imagePullSecrets:
    - name: my-org-secret
```


## spec.controller-custom-certs

To enable tag-to-digest resolution, the Knative Serving controller
needs to access the container registry, and if your registry uses a
self-signed cert, you'll need to convince the controller to trust it.

The operator encapsulates [these steps to enable tag-to-digest
resolution](https://knative.dev/development/serving/tag-resolution/).
All you have to do is create the `ConfigMap` or `Secret` and the
operator configures the serving controller for you.

These sub-fields are required:

- `name`: the name of the ConfigMap or Secret.
- `type`: either the string "ConfigMap" or "Secret".

Your `ConfigMap` or `Secret` must reside in the same namespace as your
knative resources.

For example, this spec will trigger the operator to create and mount a
volume containing the certificate in the controller and set the
required environment variable properly, assuming that certificate is
in a `ConfigMap` named `certs` in the `knative-serving` namespace.

```
metadata:
  name: knative-serving
  namespace: knative-serving
spec:
  controller-custom-certs:
    name: certs
    type: ConfigMap
```


## spec.knative-ingress-gateway

If you desire to [set up a custom ingress
gateway](https://knative.dev/development/serving/setting-up-custom-ingress-gateway/),
you can accomplish steps 2 and 3 in that doc with the following config
of the `KnativeServing` instance:

```
apiVersion: operator.knative.dev/v1alpha1
kind: KnativeServing
metadata:
  name: knative-serving
  namespace: knative-serving
spec:
  knative-ingress-gateway:
    selector:
      custom: ingressgateway
  config:
    istio:
      gateway.knative-serving.knative-ingress-gateway: "custom-ingressgateway.istio-system.svc.cluster.local"
```


## spec.cluster-local-gateway

This field enables you to use a custom local gateway with a name other
than `cluster-local-gateway`

This example shows a service and deployment `custom-local-gateway` in
the namespace `istio-system`, with the label `custom: custom-local-gw`:

```
apiVersion: operator.knative.dev/v1alpha1
kind: KnativeServing
metadata:
  name: knative-serving
  namespace: knative-serving
spec:
  cluster-local-gateway:
    selector:
      custom: custom-local-gateway
  config:
    istio:
      local-gateway.knative-serving.cluster-local-gateway: "custom-local-gateway.istio-system.svc.cluster.local"
```


## spec.high-availability

By default, Knative Serving runs a single instance of each controller.
This field allows you to configure the number of replicas for the
following master-elected controllers: `controller`, `autoscaler-hpa`,
and `networking-istio`, as well as the `HorizontalPodAutoscaler`
resources for the data plane (`activator`):

The following configuration specifies a replica count of 3 for the
controllers and a minimum of 3 activators (which may scale higher if
needed):

```
apiVersion: operator.knative.dev/v1alpha1
kind: KnativeServing
metadata:
  name: knative-serving
  namespace: knative-serving
spec:
  high-availability:
    replicas: 3
```


## spec.resources

This field enables you to override the default resource settings for
the knative containers. It essentially maps container names to
[Kubernetes resource
settings](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#resource-requests-and-limits-of-pod-and-container).

The following example configures both the `activator` and `autoscaler`
to request 0.3 CPU and 100MB of RAM, and sets hard limits of 1 CPU,
250MB RAM, and 4GB of local storage:

```
spec:
  resources:
  - container: activator
    requests:
      cpu: 300m
      memory: 100Mi
    limits:
      cpu: 1000m
      memory: 250Mi
      ephemeral-storage: 4Gi
  - container: autoscaler
    requests:
      cpu: 300m
      memory: 100Mi
    limits:
      cpu: 1000m
      memory: 250Mi
      ephemeral-storage: 4Gi
```


## spec.defaultBrokerClass

Knative Eventing allows you to define a default broker class when the
user does not specify one. The operator ships with two broker classes:
`ChannelBasedBroker` and `MTChannelBasedBroker`. This field indicates
which one to use, defaulting to `ChannelBasedBroker` if not set.

```
apiVersion: operator.knative.dev/v1alpha1
kind: KnativeEventing
metadata:
  name: knative-eventing
  namespace: knative-eventing
spec:
  defaultBrokerClass: MTChannelBasedBroker
```
