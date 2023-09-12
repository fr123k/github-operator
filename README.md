[![CircleCI](https://circleci.com/gh/fr123k/github-operator/tree/master.svg?style=svg)](https://circleci.com/gh/fr123k/github-operator/tree/master)
[![DockerHub](https://img.shields.io/badge/dockerhub-fr123k%2Fgithub--operator-blue?style=plastic
)](https://hub.docker.com/r/fr123k/github-operator)
# github-operator

This kubernetes operator watches from k8s manifests `GithubSecret` and then read the specified secrets from
Google Cloud Security Manager and stores them in Github Repository Secrets.

> 📔 **Note**
> It only support Github DependaBot secrets for now.

## Description

This operator helps to setup Github Action Secrets by reading them from Google Cloud Secret Manager and store them as
Github Action Secrets.

That enables automatic management of Github Action Secrets from within Kubernetes. Can then also be integrated into the flinkit microservice workflow.

## Packaging

### Helm

```sh
make helm-template
cp -fv helm/templates/ ../github-operator-infra/shared/workload/templates/
```

## Getting Started

You’ll need a Kubernetes cluster to run against. You can use [Minikube](https://minikube.sigs.k8s.io/docs/start/) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Starting minikube

1. Start minikube
```sh
minikube start
```

2. Enable [GCP Authentication](https://minikube.sigs.k8s.io/docs/handbook/addons/gcp-auth/)
```sh
minikube addons enable gcp-auth
```

3. Using minikube docker daemon
```sh
eval $(minikube docker-env)
```

### Running on the minikube cluster

1. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/secret_v1alpha1_githubsecret.yaml
```

2. Build your image:

```sh
make docker-build
```

3. Deploy the controller to the cluster:

```sh
make deploy
```

4. Create needed K8s Secret
```sh
kube create secret generic github-operator-secrets --from-literal=GITHUB_TOKEN=insert_the_token_here
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller from the cluster:

```sh
make undeploy
```

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

## Related Articles

### Minikube

[gcp-auth](https://www.zhaowenyu.com/minikube-doc/addons/gcp-auth.html)

### Kubernetes Operators

[hello-world-tutorial-with-kubernetes-operators](https://developers.redhat.com/blog/2020/08/21/hello-world-tutorial-with-kubernetes-operators)
[Building Operators with Golang](https://sdk.operatorframework.io/docs/building-operators/golang/)
[advanced-topics](https://sdk.operatorframework.io/docs/building-operators/golang/advanced-topics/)
[handle-cleanup-on-deletion](https://sdk.operatorframework.io/docs/building-operators/golang/advanced-topics/#handle-cleanup-on-deletion)
[manage-cr-status-conditions](https://sdk.operatorframework.io/docs/building-operators/golang/advanced-topics/#manage-cr-status-conditions)

### ArgoCD

[ArgoCD Resource Health Checks](https://argo-cd.readthedocs.io/en/stable/operator-manual/health/#way-1-define-a-custom-health-check-in-argocd-cm-configmap)
[ArgoCD Resource Health Checks](https://argo-cd.readthedocs.io/en/stable/operator-manual/health/)
