# HwOperator

Kubernetes operator written with operator-sdk in Go.

## Requirements

* An accessible Kubernetes cluster
  * Local version used: [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/)
* kubectl CLI tool
* [ingress-nginx](https://kubernetes.github.io/ingress-nginx/) addon
* [cert-manager](https://cert-manager.io/docs/installation/) addon
* ClusterIssuer for Letsencrypt support
  * run `kubectl apply -n cert-manager -f cluster-issuer.yaml`

## Poor man's installation

1. Download repo
2. Follow some tutorials to install the toolchain
3. Run `make deploy`

### Operator usage example

Create an object with the HwOperator crd like in the following example:

```yaml
apiVersion: hwoperator.com/v1
kind: HwOperator
metadata:
  name: hwoperator-sample
spec:
  replicas: 2
  host: hwoperator.example.com
  image: "nginx:latest"
```

### Options

* **replicas**: number of instances to run (managed by Deployment object)
* **host**: host name of the service you want to run
* **image**: docker image of the service you want to run

## Known "bugs"

* Can't access website via :80,:443 (can access through nodePort though)
  * I should have listened to [you guys](https://banzaicloud.com/blog/kind-ingress)? Maybe? Maybe all the links are dead. :(
  * Should have used minikube
* cert-manager/webhook doesn't care about my TLS-ready Ingress objects
