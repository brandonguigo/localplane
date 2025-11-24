# Overview

k8s-local-bench is a small CLI tool to create and manage a local Kubernetes cluster intended for development and testing. The project uses `kind` (Kubernetes-in-Docker) and includes a lightweight load-balancer helper to support local service routing.

Quick run (development):

```bash
go run main.go <command> [flags]
```

Build and run:

```bash
go build -o k8s-local-bench ./...
./k8s-local-bench <command> [flags]
```

Project layout (relevant files):

- `cmd/` — cobra commands and entry points (`root`, `cluster`, `create`, `destroy`).
- `config/` — `config.CliConfig` struct and package-level config instance.
- `utils/kind/` — helpers to create clusters and manage the load balancer.
- `cluster-config.yaml` — sample kind cluster config at repository root.

Read `configuration.md` next for detailed configuration and environment variable behavior.
