package create

import (
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"k8s-local-bench/utils/kubectl"
)

// applyBootstrapManifests applies the bootstrap manifests from the local-argo
// chart into the created cluster. It logs and fatally exits on failure.
func applyBootstrapManifests(cmd *cobra.Command, kubeconfigPath string, base string) {
	kubectlClient := kubectl.NewClient(&kubeconfigPath, nil)
	bootstrapPath := filepath.Join(base, "local-argo", "charts", "local-stack", "bootstrap")
	patterns := []string{filepath.Join(bootstrapPath, "argo-bootstrap-*.yaml")}
	log.Info().Strs("patterns", patterns).Msg("applying bootstrap manifests into cluster")
	if err := kubectlClient.ApplyPaths(cmd.Context(), patterns); err != nil {
		log.Fatal().Err(err).Msg("failed to apply bootstrap manifests into cluster")
	} else {
		log.Info().Msg("applied bootstrap manifests into cluster")
	}
}
