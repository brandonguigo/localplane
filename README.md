> [!IMPORTANT]
> **localplane has moved to GitLab.**
>
> This repository is archived and read-only. Development continues at
> **<https://gitlab.com/atomic-blend/kubernetes/localplane/localplane>**,
> as part of the [Atomic Blend](https://gitlab.com/atomic-blend) project.
>
> Issues and pull requests here are no longer monitored — please open them on GitLab instead.
> Releases, including prebuilt binaries for macOS and Linux, are published there:
> <https://gitlab.com/atomic-blend/kubernetes/localplane/localplane/-/releases>
>
> The content below is preserved as it was at the time of the move and is no longer updated.

# localplane

## Quickstart

Create a workspace and run the CLI to create a local `kind` cluster. The `create` command performs some helpful setup by default (local-argo repo, ArgoCD installation, bootstrap manifests):

```bash
mkdir my-local-workspace
export LOCALPLANE_DIRECTORY=$PWD
localplane cluster create
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