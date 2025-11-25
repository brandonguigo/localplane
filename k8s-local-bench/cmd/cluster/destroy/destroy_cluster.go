package destroy

import (
	"os/exec"
	"strings"
	"time"

	"k8s-local-bench/cmd/cluster/shared"
	"k8s-local-bench/config"
	kindsvc "k8s-local-bench/utils/kind"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func destroyCluster(cmd *cobra.Command, args []string) {
	log.Info().Msg("Deleting local k8s cluster...")
	if config.CliConfig.Debug {
		log.Debug().Bool("debug", true).Msg("debug enabled")
	}

	clusterName, _ := cmd.Flags().GetString("cluster-name")
	// if no cluster name provided, list existing kind clusters and ask user to pick one
	if strings.TrimSpace(clusterName) == "" {
		sel, err := selectClusterInteractive()
		if err != nil {
			log.Info().Msg("no kind clusters found")
			return
		}
		clusterName = sel
	}

	kindCfg := shared.FindKindConfig(clusterName)
	if kindCfg == "" {
		log.Info().Msg("no kind config file found in current directory; proceeding without one")
	} else {
		log.Info().Str("path", kindCfg).Msg("found kind config file in current directory")
	}

	// shutdown cluster
	kindsvcClient := kindsvc.NewClient("")
	if err := kindsvcClient.Delete(clusterName); err != nil {
		log.Error().Err(err).Msg("failed deleting kind cluster")
	} else {
		log.Info().Str("name", clusterName).Msg("kind cluster deletion invoked")
	}

	// attempt to stop any running cloud-provider-kind process (best-effort)
	if out, err := exec.Command("pkill", "-f", "sudo cloud-provider-kind").CombinedOutput(); err != nil {
		log.Debug().Err(err).Str("output", string(out)).Msg("pkill for cloud-provider-kind returned error (may be fine if not running)")
	} else {
		log.Info().Msg("stopped cloud-provider-kind processes")
	}

	// make sure the cluster is stopped/deleted: poll `kind get clusters` briefly
	const maxAttempts = 6
	if !waitForClusterStopped(clusterName, maxAttempts, 1*time.Second) {
		log.Warn().Str("name", clusterName).Msg("cluster still present after deletion attempts; manual cleanup may be needed")
		return
	}

	// cleanup local files
	cleanup(clusterName)
}
