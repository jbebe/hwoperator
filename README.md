## Requirements

* kubernetes cluster
* kubectl
* ingress-nginx
  * https://kubernetes.github.io/ingress-nginx/deploy/#bare-metal
  * kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v0.48.1/deploy/static/provider/baremetal/deploy.yaml

* cert-manager
  * https://cert-manager.io/docs/installation/
  * kubectl apply -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml