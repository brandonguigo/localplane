# k8s-local-bench CLI

This document describes the `k8s-local-bench` command-line interface, how configuration is loaded, and the two cluster-related commands implemented in this repository.

## Overview

- Binary: `k8s-local-bench` (development: run with `go run main.go`)
- Purpose: create and manage a local Kubernetes cluster (using `kind` and a small load-balancer helper).

## Installation / Run (quick)

Development (no build step):

```bash
go run main.go <command> [flags]
```

Build a binary:

```bash
go build -o k8s-local-bench ./...
./k8s-local-bench <command> [flags]
```

## Configuration

The CLI supports configuration via (in precedence order): command-line flags, environment variables, and an optional config file.

- Config file flag: `--config, -c` — if provided, the specified file is used.
- Default config file: a hidden file named `.k8s-local-bench` (YAML) is searched for in `$HOME` if `--config` is not provided.
- Environment variables: prefixed with `K8S_LOCAL_BENCH` (e.g. `K8S_LOCAL_BENCH_DIRECTORY`).
- Flags are bound to Viper and can be set on the CLI; root persistent flags include `--directory` (`-d`).

Config fields (unmarshalled into `config.CliConfig`):

- `debug` (bool): enable debug logging (can also be set via `LOG_LEVEL=debug`).
- `directory` (string): directory where configurations and data are stored. This is used by commands to look for cluster-specific config files (e.g. `clusters/<name>/kind-config.yaml`).

Example env usage:

```bash
export K8S_LOCAL_BENCH_DIRECTORY=/path/to/configs
export K8S_LOCAL_BENCH_DEBUG=true
go run main.go cluster create
```

## Project-provided Kind config example

There is a sample kind config at the repository root named `cluster-config.yaml`. A minimal example looks like:

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
```

Commands may also search for files named `kind-config.yaml`, `kind.yaml`, or similar under the configured `directory` and under `clusters/<cluster-name>`.

## Commands

Top-level command: `k8s-local-bench`.

Subcommand group: `cluster` — control local k8s clusters.

Persistent flags available to the `cluster` command and its subcommands:

- `--cluster-name` (string) — default: `local-bench`. Name of the cluster and the directory name under `clusters/` the tool will use.

### cluster create

Usage:

```bash
k8s-local-bench cluster create [flags]
```

What it does:

- Creates a local `kind` cluster (the implementation uses the `utils/kind` helper).
- Looks for a kind config file in the following order:
  - `$(directory)/clusters/<cluster-name>/kind-config.yaml` (or `kind-config.yml`, or glob `kind*.y*ml`)
  - `$(directory)/kind-config.yaml` (or `kind*.y*ml`)
  - CWD `kind-config.yaml` (or `kind*.y*ml`)

Flags specific to `create`:

- `-y, --yes` (bool): don't ask for confirmation; assume yes.
- `--start-lb` (bool, default: true): start the local load balancer (cloud-provider-kind helper).
- `--lb-foreground` (bool, default: false): run load balancer in the foreground (blocking); otherwise it runs in background.

Examples:

```bash
# Run interactively and allow confirmation
go run main.go cluster create

# Provide directory and auto-confirm
go run main.go cluster create -d ../tmp -y

# Don't start the built-in load balancer
./k8s-local-bench cluster create --start-lb=false

# Run load balancer in foreground (blocking)
./k8s-local-bench cluster create --lb-foreground
```

Notes:

- The command references `config.CliConfig.Debug` and respects the global debug setting.
- After cluster creation the tool attempts to start a simple load balancer helper. The background/foreground behavior is controlled with `--lb-foreground`.

### cluster destroy

Usage:

```bash
k8s-local-bench cluster destroy [flags]
```

What it does:

- Intended to delete/stop a local `kind` cluster.
- The implementation currently looks for kind config files in the same search locations as `create`.

Flags:

- Inherits `--cluster-name` from `cluster` persistent flags.

Status / caveats:

- The `destroy` command implementation contains TODOs in the codebase: it logs intent and locates config files but the actual delete/stop logic is not yet implemented. Treat `destroy` as a placeholder until the removal routines are implemented.

## Examples & common workflows

Create a cluster using a config stored under the CLI `directory`:

```bash
export K8S_LOCAL_BENCH_DIRECTORY=$PWD
go run main.go cluster create -y
```

Create a cluster with an explicit config file and run the load balancer in foreground:

```bash
go run main.go cluster create -y --lb-foreground
```

Destroy (note: destroy currently has TODOs and may not remove the cluster fully):

```bash
go run main.go cluster destroy --cluster-name local-bench
```

## Next steps / developer notes

- If you want `cluster destroy` to be available from the CLI, ensure it is added to the `cluster` command (the code for `destroy` exists under `cmd/cluster/destroy` but may not be wired into `cmd/cluster/root.go`).
- Consider implementing cluster status checks and more robust waiting logic after `kind` creation to confirm the cluster is ready.

---

If you want, I can also:
- add examples directly to the repository `README.md`, or
- wire `destroy` into the `cluster` root command so it's available at runtime.
