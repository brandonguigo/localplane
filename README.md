# k8s-local-bench

## Setup a k8s cluster to work locally

```
kind create cluster --config cluster-config.yaml --kubeconfig ~/.kube/local-kind
```

### Cluster configuration
TODO: refactor the doc to use dnsmasq (macOS example for now following the gist tutorial)
To avoid hassles with ingress and local DNS, the easiest way to expose a few services to the host is with the host port mapping.
`cluster-config.yaml` defines a set of nodeports that are made available to the host (so localhost access works).

## Install CloudNative PG Operator
```
helm repo add cnpg https://cloudnative-pg.github.io/charts
helm upgrade --install cnpg \
  --namespace cnpg-system \
  --create-namespace \
  cnpg/cloudnative-pg
```

# TODO
- [ ] update the documentation on all the new features / concepts
- [ ] add mermaid diagrams explaining how it works
- [ ] test ingress with dnsmasq manually (lookup the cloud-provider-kind logs + update ip in $(brew --prefix)/etc/dnsmasq.conf)
- [ ] test to deploy a httpbin as a workbench app
- [ ] add httpbin deployment in the k8s-local-bench chart