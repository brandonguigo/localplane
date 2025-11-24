# Configuration (in-depth)

This page describes how `k8s-local-bench` reads configuration, the precedence rules, and how flags, environment variables, and config files work together.

Config sources and precedence (highest → lowest):

1. Command-line flags
2. Environment variables (prefixed with `K8S_LOCAL_BENCH`)
3. Explicit config file passed with `--config` / `-c`
4. Default config file `.k8s-local-bench` found in `$HOME` (YAML)

Key implementation points:

- The CLI uses Cobra for flags and Viper for configuration. Flags are bound to Viper during `initializeConfig` so both flags and environment variables are available to commands.
- Root persistent flags include `--directory, -d` (CLI configuration/data directory) and `--config, -c` (explicit config file path).
- Viper is configured with `viper.SetEnvPrefix("K8S_LOCAL_BENCH")` and a replacer so nested keys or dashes are available as underscored env vars (e.g. `K8S_LOCAL_BENCH_DIRECTORY`).

Config structure (`config.Config`):

- `Debug` (bool): enables debug-level logging (also toggled by `LOG_LEVEL=debug`).
- `Directory` (string): the base directory the CLI uses to locate supplemental config, clusters, and data.

Config file behavior:

- If `--config` is provided, Viper will use that exact file path.
- If not provided, Viper will look for `$HOME/.k8s-local-bench.yaml` (or other supported extensions) via `viper.SetConfigName(".k8s-local-bench")` and `viper.AddConfigPath("$HOME")`.
- Missing config file is not an error; the CLI continues using flags and environment variables.

Environment variables:

- Use `K8S_LOCAL_BENCH_<UPPER_KEY>` form. Nested keys replace `.` and `-` with `_`.
- Examples:

```bash
export K8S_LOCAL_BENCH_DIRECTORY=/path/to/configs
export K8S_LOCAL_BENCH_DEBUG=true
```

Flag binding notes:

- The code binds both local and persistent flags to Viper, and also binds parent persistent flags to ensure subcommands see root-level flags.
- This allows using `--directory` at the top-level or on subcommands and have Viper pick it up during `PersistentPreRunE`.

Where commands look for `kind` config files:

- Commands use a helper `findKindConfig(clusterName)` (present in both `create` and `destroy` command files). The search order implemented by that function is:
  1. If `cluster-name` is provided, look under `$(directory)/clusters/<cluster-name>/` for `kind-config.yaml`, `kind-config.yml`, or glob `kind*.y*ml`.
  2. Look under the CLI `directory` root for `kind-config.yaml`/`kind*.y*ml`.
  3. Fallback to the current working directory and search for the same file names/globs.

Example `.k8s-local-bench` YAML config (in `$HOME`):

```yaml
debug: false
directory: /home/you/.k8s-local-bench
```

Troubleshooting:

- If a command doesn't seem to see your `directory` value, verify the `--directory` flag usage or export `K8S_LOCAL_BENCH_DIRECTORY` before running the command.
- To force a particular config file, supply `--config /path/to/file.yaml`.
