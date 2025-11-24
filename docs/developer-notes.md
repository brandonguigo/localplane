# Developer Notes

This page contains actionable items for contributors who want to extend or harden the CLI.

Wiring `destroy` into the `cluster` command:

- File: `cmd/cluster/root.go`
- Add the `destroy` subcommand alongside `create`:

```go
import (
    "k8s-local-bench/cmd/cluster/create"
    "k8s-local-bench/cmd/cluster/destroy"
)

cmd.AddCommand(create.NewCommand())
cmd.AddCommand(destroy.NewCommand())
```

Implementing `destroy` functionality:

- Extend `utils/kind` with a `Delete(name)` helper that calls `kind delete cluster --name <name>` and returns structured errors.
- Add a `StopLoadBalancer(name)` helper to stop any background load-balancer processes (ensure `StartLoadBalancer` records enough state to stop it later).

Testing suggestions:

- Add integration tests that use `kind` to create and delete ephemeral clusters (mark them as long-running or behind a build tag).
- Unit test `findKindConfig` by exercising its search order using a temporary directory layout.

Other improvements:

- Add cluster readiness checks after create: wait for `kubectl get nodes` to show control-plane ready and for core DNS to be available.
- Consider adding `--kubeconfig` support or outputting the `KUBECONFIG` path used for the created cluster.
- Improve logging of background process PIDs and provide log files for the load-balancer helper.
