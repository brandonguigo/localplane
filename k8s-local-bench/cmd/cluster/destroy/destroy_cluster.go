package destroy

import (
	"fmt"
	"strings"
	"time"

	"k8s-local-bench/cmd/cluster/shared"
	"k8s-local-bench/config"
	kindsvc "k8s-local-bench/utils/kind"

	"github.com/manifoldco/promptui"
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

	// confirm deletion with the user
	prompt := promptui.Select{
		Label: fmt.Sprintf("Are you sure you want to delete kind cluster '%s'?", clusterName),
		Items: []string{"No", "Yes"},
		Size:  2,
	}
	i, _, err := prompt.Run()
	if err != nil {
		log.Error().Err(err).Msg("confirmation prompt failed")
		return
	}
	if i != 1 { // user chose "No"
		log.Info().Str("name", clusterName).Msg("cluster deletion cancelled by user")
		return
	}

	// shutdown cluster
	kindsvcClient := kindsvc.NewClient("")
	if err := kindsvcClient.Delete(clusterName); err != nil {
		log.Error().Err(err).Msg("failed deleting kind cluster")
	} else {
		log.Info().Str("name", clusterName).Msg("kind cluster deletion invoked")
	}

	// stop the load balancer provider if running
	if err := kindsvcClient.StopLoadBalancer(clusterName); err != nil {
		log.Warn().Err(err).Str("name", clusterName).Msg("failed to stop cloud-provider-kind load balancer (it may not have been running)")
	} else {
		log.Info().Str("name", clusterName).Msg("stopped cloud-provider-kind load balancer (if it was running)")
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
