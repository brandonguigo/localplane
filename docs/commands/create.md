# cluster create — Detailed

Location: `cmd/cluster/create/root.go`

Purpose:

- Create a local `kind` cluster and optionally start a small load-balancer helper.

Usage:

```bash
k8s-local-bench cluster create [flags]
```

Flags:

- `-y, --yes` (bool): skip interactive confirmation and proceed.
- `--start-lb` (bool, default: true): whether to start the local load balancer helper.
- `--lb-foreground` (bool, default: false): if true, run the load balancer in the foreground (blocking); if false, it runs in the background.
- inherited: `--cluster-name` (default `local-bench`), `--directory` (root CLI directory)

High-level flow (implementation notes):

1. Logs an informational message: "Creating local k8s cluster...".
2. Honors `config.CliConfig.Debug` to enable debug logging inside the command.
3. Retrieves `--cluster-name` and calls `findKindConfig(clusterName)` to locate a kind configuration file. The function searches (in order):
   - `$(directory)/clusters/<cluster-name>/kind-config.yaml|yml` and glob `kind*.y*ml`
   - `$(directory)/kind-config.yaml|yml` and glob `kind*.y*ml`
   - CWD `kind-config.yaml|yml` and glob `kind*.y*ml`
   If found, its absolute path is logged and passed into the kind helper.
4. Asks for confirmation unless `--yes` is provided.
5. Calls `kindsvc.Create(clusterName, kindCfg)` from `utils/kind` to create the kind cluster. Any errors are logged.
6. If `--start-lb` is true, calls `kindsvc.StartLoadBalancer(clusterName, background)` where `background` is `true` if `--lb-foreground` is false. Errors are logged.

Notes about `utils/kind` responsibilities (refer to `utils/kind/kind.go`):

- `Create(name, kindConfigPath)` should encapsulate invoking `kind` to create a cluster. It may accept an empty config path to use default behavior.
- `StartLoadBalancer(name, background)` should start the cloud-provider-kind process (background vs foreground behavior).

Examples:

```bash
# Interactive create (asks for confirmation)
go run main.go cluster create

# Non-interactive, use alternate directory and cluster name
go run main.go cluster create -d ../tmp --cluster-name test-cluster -y

# Create and run load balancer in foreground
./k8s-local-bench cluster create --lb-foreground
```

Testing and verification tips:

- After `kindsvc.Create` returns, verify the cluster with `kubectl cluster-info --context kind-<cluster-name>`.
- Check load-balancer logs if running in foreground; for background mode, the helper should log process start and PID to the configured output.
