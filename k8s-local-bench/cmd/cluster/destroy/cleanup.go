package destroy

import (
	"k8s-local-bench/config"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

func cleanup(clusterName string) {
	kubeconfigPath := filepath.Join(config.CliConfig.Directory, "clusters", clusterName, "kubeconfig")

	if err := os.Remove(kubeconfigPath); err != nil {
		log.Warn().Err(err).Str("path", kubeconfigPath).Msg("failed to delete kubeconfig file")
	} else {
		log.Info().Str("path", kubeconfigPath).Msg("deleted kubeconfig file")
	}
}
