#!/bin/bash

set -eo pipefail
OS=$(uname -s | tr A-Z a-z)
ARCH=$(uname -m | sed 's/aarch64/arm64/' | sed 's/x86_64/amd64/')

function install_kubectl() {
  curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/$OS/$ARCH/kubectl"
  chmod +x kubectl
  sudo mv kubectl /usr/local/bin/kubectl
}

function install_helm() {
  local HELM_VERSION=v3.17.1
  curl -OL https://get.helm.sh/helm-$HELM_VERSION-$OS-$ARCH.tar.gz
  tar -xvf helm-$HELM_VERSION-$OS-$ARCH.tar.gz
  chmod +x $OS-$ARCH/helm
  sudo mv $OS-$ARCH/helm /usr/local/bin/helm
  rm -rf helm-$HELM_VERSION-$OS-$ARCH*
}

install_kubectl

kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.0/experimental-install.yaml

install_helm

function install_istio() {
  local ISTIO_VERSION=1.24.3

  # curl -L https://github.com/istio/istio/releases/download/$ISTIO_VERSION/istioctl-$ISTIO_VERSION-$(uname -s | tr A-Z a-z)-$(uname -m).tar.gz -o istioctl.tar.gz
  curl -L https://github.com/istio/istio/releases/download/$ISTIO_VERSION/istio-$ISTIO_VERSION-$(uname -s | tr A-Z a-z | sed 's/darwin/osx/')-$(uname -m).tar.gz -o istioctl.tar.gz

  tar -xvf istioctl.tar.gz
  sudo mv istio-$ISTIO_VERSION/bin/istioctl /usr/local/bin/istioctl
  rm -rf istioctl.tar.gz istio-$ISTIO_VERSION

  istioctl install --set profile=demo --skip-confirmation --verify

  kubectl -n istio-system set env deploy/istiod PILOT_ENABLE_ALPHA_GATEWAY_API=true

  cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
 name: istio
spec:
 controllerName: istio.io/gateway-controller
EOF
}

function install_cert_manager() {
  local CERT_MANAGER_VERSION=v1.17.1

  helm repo add jetstack https://charts.jetstack.io --force-update

  helm install cert-manager jetstack/cert-manager --version $CERT_MANAGER_VERSION \
    --namespace cert-manager --create-namespace \
    --set crds.enabled=true \
    --set config.apiVersion="controller.config.cert-manager.io/v1alpha1" \
    --set config.kind="ControllerConfiguration" \
    --set config.enableGatewayAPI=true

  kubectl wait --namespace cert-manager \
    --for=condition=ready pod \
    --selector=app.kubernetes.io/instance=cert-manager \
    --timeout=120s

  cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
 name: self-signed
spec:
 selfSigned: {}
EOF

#   # Create letsencrypt-staging ClusterIssuer
#   cat <<EOF | kubectl apply -f -
# apiVersion: cert-manager.io/v1
# kind: ClusterIssuer
# metadata:
#   name: letsencrypt-staging
# spec:
#   acme:
#     email: security@example.org  ## <---------------------------- REPLACE EMAIL AND UNCOMMENT
#     server: https://acme-staging-v02.api.letsencrypt.org/directory
#     privateKeySecretRef:
#       name: letsencrypt-staging-clusterissuer-account-key
#     solvers:
#     - http01:
#         ingress:
#           ingressClassName: istio
# EOF
#   # Create letsencrypt-prod ClusterIssuer
#   cat <<EOF | kubectl apply -f -
# apiVersion: cert-manager.io/v1
# kind: ClusterIssuer
# metadata:
#   name: letsencrypt-prod
# spec:
#   acme:
#     email: security@example.org ## <---------------------------- REPLACE EMAIL AND UNCOMMENT
#     server: https://acme-v02.api.letsencrypt.org/directory
#     privateKeySecretRef:
#       name: letsencrypt-prod-clusterissuer-account-key
#     solvers:
#     - http01:
#         ingress:
#           ingressClassName: istio
# EOF
}

function install_envoy_gateway() {
  local ENVOY_GATEWAY_VERSION=v1.1.1
  helm install eg oci://docker.io/envoyproxy/gateway-helm --version $ENVOY_GATEWAY_VERSION -n envoy-gateway-system --create-namespace --set deployment.replicas=3
  kubectl wait --timeout=5m -n envoy-gateway-system deployment/envoy-gateway --for=condition=Available
}


# install_istio
      
install_cert_manager

install_envoy_gateway
