package create

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// NewCommand creates the cluster command
func NewCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "create",
		Short: "create a local k8s cluster",
		Run:   createCluster,
	}
	// flags
	cmd.Flags().BoolP("yes", "y", false, "don't ask for confirmation; assume yes")
	cmd.Flags().Bool("start-lb", true, "start local load balancer (cloud-provider-kind)")
	cmd.Flags().Bool("lb-foreground", false, "run load balancer in foreground (blocking)")
	cmd.Flags().Bool("disable-argocd", false, "don't perform ArgoCD related setup")
	// add subcommands here
	log.Debug().Msg("cluster create command initialized")
	return cmd
}
