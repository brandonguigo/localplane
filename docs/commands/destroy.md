# cluster destroy — Detailed

Location: `cmd/cluster/destroy/root.go`

Purpose:

- Intended to stop and remove a local `kind` cluster created by this tool.

Usage:

```bash
k8s-local-bench cluster destroy [flags]
```

Flags:

- inherited: `--cluster-name` (default `local-bench`), `--directory` (root CLI directory)

Current implementation status:

- The command currently logs intent and locates a `kind` config file using the same `findKindConfig` strategy as `create`.
- The actual cluster deletion and load-balancer shutdown logic are TODOs in the codebase; `destroy` is a placeholder until removal routines are implemented.

How `findKindConfig` searches for kind configs:

- If `cluster-name` is set: `$(directory)/clusters/<cluster-name>/kind-config.yaml|yml` and `kind*.y*ml` glob.
- Then `$(directory)/kind-config.yaml|yml` and `kind*.y*ml` glob.
- Then current working directory, same filename patterns.

Recommended implementation steps (developer guidance):

1. Use `kind` to delete the cluster: `kind delete cluster --name <cluster-name>` (or the equivalent API in the `utils/kind` helper).
2. Ensure any background `cloud-provider-kind` process started by `StartLoadBalancer` is stopped. If `StartLoadBalancer` tracked PIDs or uses a supervisor, call the stop routine.
3. Add robust error handling and status checks to confirm cluster deletion. Consider verifying that `kubectl` no longer lists the cluster nodes.

Example (expected once implemented):

```bash
# Delete cluster and associated load balancer
./k8s-local-bench cluster destroy --cluster-name local-bench
```

Developer note:

- Add unit tests for the `utils/kind` helper to simulate create/delete and background process management.
- Consider adding a `--force` flag for non-interactive deletion in the future.
