package destroy

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// NewCommand creates the cluster command
func NewCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "destroy",
		Short: "destroy a local k8s cluster",
		Run:   destroyCluster,
	}
	// add subcommands here
	log.Debug().Msg("cluster destroy command initialized")
	return cmd
}
