package create

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	kindsvc "k8s-local-bench/utils/kind"
)

// startLocalLoadBalancer reads flags from the provided cobra command and
// starts the cloud-provider-kind load balancer via the provided kind client.
func startLocalLoadBalancer(kindClient *kindsvc.Client, cmd *cobra.Command, clusterName string) {
	startLB, _ := cmd.Flags().GetBool("start-lb")
	lbFg, _ := cmd.Flags().GetBool("lb-foreground")
	if startLB {
		if !lbFg {
			if err := kindClient.StartLoadBalancer(clusterName, true); err != nil {
				log.Error().Err(err).Msg("failed to start load balancer in background")
			} else {
				log.Info().Msg("load balancer started in background")
			}
		} else {
			if err := kindClient.StartLoadBalancer(clusterName, false); err != nil {
				log.Error().Err(err).Msg("failed to run load balancer (foreground)")
			} else {
				log.Info().Msg("load balancer run completed")
			}
		}
	}
}
