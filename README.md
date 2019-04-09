# Knative Ambassador Ingress

An [Ambassador](https://getambassador.io) ingress implementation for
Knative Serving.

This is an early prototype as a way to explore alternative ingress
implementations for Knative to provide lightweight non-Istio options
for users that don't want or need Istio.

Work is in-progress to switch Knative Eventing from a hard dependency
on Istio to the same ingress abstraction Serving uses. Once
[knative/eventing#294](https://github.com/knative/eventing/issues/294)
and
[knative/eventing#918](https://github.com/knative/eventing/issues/918)
are fixed then Ambassador will also work with Eventing.

# Installation

The instructions below were tested against Knative Serving v0.5.0 and
minikube running Kubernetes v1.12.0.

## Create a Kubernetes cluster

Follow the Knative Installation instructions to [create a Kubernetes
cluster](https://www.knative.dev/docs/install/). DO NOT install Istio
or Knative into that cluster yet.

If you want to use minikube, you can start a cluster with:

```shell
minikube start --memory=8192 --cpus=4 \
  --kubernetes-version=v1.12.0 \
  --vm-driver=kvm2 \
  --disk-size=30g \
  --extra-config=apiserver.enable-admission-plugins="LimitRanger,NamespaceExists,NamespaceLifecycle,ResourceQuota,ServiceAccount,DefaultStorageClass,MutatingAdmissionWebhook"
```

## Install Ambassador

```shell
kubectl apply --filename https://github.com/bbrowning/knative-ambassador-ingress/releases/download/v0.0.1/ambassador.yaml
```

## Install Istio CRDs

While we don't need Istio when using Knative Serving with Ambassador,
for now Knative Serving always bundles the Istio-based ClusterIngress
implementation and thus we need the Istio CRDs to be present even
though we won't install Istio itself.

```shell
kubectl apply --filename https://github.com/knative/serving/releases/download/v0.5.0/istio-crds.yaml
```

## Install Knative Serving

```shell
kubectl apply --filename https://github.com/knative/serving/releases/download/v0.5.0/serving.yaml
```

## Change the default ClusterIngress class

By default Istio is used for all ingress traffic. Switch that default
to our Ambassador implementation.

```shell
kubectl patch configmap -n knative-serving config-network -p '{"data": {"clusteringress.class": "ambassador.ingress.networking.knative.dev"}}'
```

## Run the Ambassador Ingress

```shell
kubectl apply --filename https://github.com/bbrowning/knative-ambassador-ingress/releases/download/v0.0.1/release.yaml
```

## Deploy the Knative helloworld-go sample

```shell
cat <<EOF | kubectl apply -f -
apiVersion: serving.knative.dev/v1alpha1
kind: Service
metadata:
  name: helloworld-go
  namespace: default
spec:
  runLatest:
    configuration:
      revisionTemplate:
        spec:
          container:
            image: gcr.io/knative-samples/helloworld-go
            env:
              - name: TARGET
                value: "Go Sample v1"
EOF
```

# Testing changes locally

If you want to hack on this Ambassador ingress implementation, clone
the repo and run the controller locally:

```shell
AMBASSADOR_NAMESPACE="ambassador" WATCH_NAMESPACE="" go run cmd/manager/main.go
```

# Building, pushing, and testing changes

This is how I do it, at least. You'll need to change the repos to ones
that aren't bbrowning.

```shell
operator-sdk build quay.io/bbrowning/knative-ambassador-ingress:v0.0.1
docker push quay.io/bbrowning/knative-ambassador-ingress:v0.0.1
```

Update the image in deploy/release.yaml and tag the git repo with the
same version as the image.
