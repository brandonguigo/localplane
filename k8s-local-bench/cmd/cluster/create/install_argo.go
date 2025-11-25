package create

import (
	"github.com/rs/zerolog/log"

	argocdsvc "k8s-local-bench/utils/argocd"
)

// installArgoIfRequested installs or upgrades ArgoCD via Helm when
// not disabled. It logs and fatally exits on errors (preserving previous behavior).
func installArgoIfRequested(kubeconfigPath string, disableArgoCD bool) {
	if disableArgoCD {
		log.Info().Msg("Argocd setup disabled; skipping ArgoCD related tasks")
		return
	}

	mounts := []argocdsvc.RepoMount{{
		Name:      "local-argo",
		HostPath:  "/mnt/local-argo",
		MountPath: "/mnt/local-argo",
	}}
	argocdsvcClient := argocdsvc.NewClient(kubeconfigPath)
	out, err := argocdsvcClient.InstallOrUpgradeArgoCD(mounts)
	if err != nil {
		log.Fatal().Err(err).Str("output", out).Msg("failed to install argocd via helm sdk")
	} else {
		log.Info().Str("output", out).Msg("argocd installed")
	}
}
