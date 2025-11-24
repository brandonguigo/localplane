package destroy

import (
	"os"
	"path/filepath"

	"k8s-local-bench/config"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// NewCommand creates the cluster command
func NewCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "destroy",
		Short: "destroy a local k8s cluster",
		Run:   createCluster,
	}
	// add subcommands here
	return cmd
}

func createCluster(cmd *cobra.Command, args []string) {
	log.Info().Msg("Deleting local k8s cluster...")
	// honor CLI debug config if set
	if config.CliConfig.Debug {
		log.Debug().Bool("debug", true).Msg("debug enabled")
	}

	clusterName, _ := cmd.Flags().GetString("cluster-name")
	// check for kind config file (looks inside CLI config directory clusters/<cluster-name>)
	kindCfg := findKindConfig(clusterName)
	if kindCfg == "" {
		log.Info().Msg("no kind config file found in current directory; proceeding without one")
	} else {
		log.Info().Str("path", kindCfg).Msg("found kind config file in current directory")
	}

	// TODO: delete a kind cluster (name is currently fixed)

	// TODO: stop the cloud-provider-kind command

	// TODO: make sure the cluster is stopped/deleted
}

// findKindConfig searches the current working directory for common kind config filenames.
// Returns the first match (absolute path) or empty string if none found.
func findKindConfig(clusterName string) string {
	base := config.CliConfig.Directory
	var err error
	if base == "" {
		base, err = os.Getwd()
		if err != nil {
			return ""
		}
	}
	candidates := []string{"kind-config.yaml", "kind-config.yml", "kind.yaml", "kind.yml"}

	// 1) cluster-specific dir under CLI config
	if clusterName != "" {
		clusterPath := filepath.Join(base, "clusters", clusterName)
		for _, name := range candidates {
			p := filepath.Join(clusterPath, name)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		matches, _ := filepath.Glob(filepath.Join(clusterPath, "kind*.y*ml"))
		if len(matches) > 0 {
			return matches[0]
		}
	}

	// 2) CLI config dir root
	for _, name := range candidates {
		p := filepath.Join(base, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	matches, _ := filepath.Glob(filepath.Join(base, "kind*.y*ml"))
	if len(matches) > 0 {
		return matches[0]
	}

	// 3) fallback to current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	for _, name := range candidates {
		p := filepath.Join(cwd, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	matches, _ = filepath.Glob(filepath.Join(cwd, "kind*.y*ml"))
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}
