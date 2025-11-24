# Examples & Common Workflows

This page shows runnable examples for common scenarios.

1) Create a cluster using repository root config

```bash
# Use the repo root as the CLI directory so the included cluster-config.yaml is used
export K8S_LOCAL_BENCH_DIRECTORY=$(pwd)
go run main.go cluster create -y
```

2) Create a cluster using a named cluster config stored under the CLI directory

```bash
# Example: ~/.k8s-local-bench/clusters/mytest/kind-config.yaml
export K8S_LOCAL_BENCH_DIRECTORY=$HOME/.k8s-local-bench
go run main.go cluster create --cluster-name mytest -y
```

3) Create but don't start the load balancer

```bash
./k8s-local-bench cluster create --start-lb=false -y
```

4) Run the load balancer in foreground for debugging

```bash
./k8s-local-bench cluster create --lb-foreground
```

5) Destroying a cluster (placeholder; may not yet remove everything)

```bash
./k8s-local-bench cluster destroy --cluster-name local-bench
```

Kind config sample (repository `cluster-config.yaml`):

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  # You can add extraPortMappings to expose container ports to host
```

Verification commands after creation:

```bash
kubectl cluster-info --context kind-local-bench
kubectl get nodes
```
