# k8s-local-bench

## Quick start — create a local cluster

Create a workspace and run the CLI to create a local `kind` cluster. The `create` command performs some helpful setup by default (local-argo repo, ArgoCD installation, bootstrap manifests):

```bash
mkdir my-local-workspace
export K8S_LOCAL_BENCH_DIRECTORY=$PWD
go run main.go cluster create
```

Common quick options:

- `-y` or `--yes`: skip interactive confirmation and proceed.
- `--cluster-name <name>`: set the cluster name (when omitted `create` prompts and defaults to `local-bench`).
- `--start-lb=false`: disable startup of the built-in cloud-provider-kind load balancer.
- `--lb-foreground`: run the load balancer in the foreground (blocking).
- `--disable-argocd`: skip ArgoCD/local-argo setup and ArgoCD Helm install.

See `docs/CLI.md` and `docs/commands/*` for more details.

### TODO

- [ ] rename to project to localplane