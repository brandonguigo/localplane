package create

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"k8s-local-bench/config"
	kindsvc "k8s-local-bench/utils/kind"

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
	// add subcommands here
	return cmd
}

func createCluster(cmd *cobra.Command, args []string) {
	log.Info().Msg("Creating local k8s cluster...")
	// honor CLI debug config if set
	if config.CliConfig.Debug {
		log.Debug().Bool("debug", true).Msg("debug enabled")
	}

	// get cluster name and locate kind config inside CLI config clusters/<name>
	clusterName, _ := cmd.Flags().GetString("cluster-name")
	// check for kind config file (looks inside CLI config directory clusters/<cluster-name>)
	kindCfg := findKindConfig(clusterName)
	if kindCfg == "" {
		log.Info().Msg("no kind config file found in current directory; proceeding without one")
	} else {
		log.Info().Str("path", kindCfg).Msg("found kind config file in current directory")
	}

	// ask for confirmation unless user passed --yes
	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		fmt.Printf("Proceed to create kind cluster '%s'? (y/N): ", "local-bench")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if !(strings.EqualFold(input, "y") || strings.EqualFold(input, "yes")) {
			log.Info().Msg("aborting cluster creation")
			return
		}
	}

	// create a kind cluster using provided cluster name
	if err := kindsvc.Create(clusterName, kindCfg); err != nil {
		log.Error().Err(err).Msg("failed creating kind cluster")
		return
	}
	log.Info().Str("name", clusterName).Msg("kind cluster creation invoked")
	// start load balancer if requested (defaults: start and run in background)
	startLB, _ := cmd.Flags().GetBool("start-lb")
	lbFg, _ := cmd.Flags().GetBool("lb-foreground")
	if startLB {
		if !lbFg {
			// background
			if err := kindsvc.StartLoadBalancer(clusterName, true); err != nil {
				log.Error().Err(err).Msg("failed to start load balancer in background")
			} else {
				log.Info().Msg("load balancer started in background")
			}
		} else {
			// foreground: run and wait (this will block until the process exits)
			if err := kindsvc.StartLoadBalancer(clusterName, false); err != nil {
				log.Error().Err(err).Msg("failed to run load balancer (foreground)")
			} else {
				log.Info().Msg("load balancer run completed")
			}
		}
	}

	// TODO: make sure the cluster is up and running via kubectl commands
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

	// 1) look in CLI config dir under clusters/<clusterName>
	if clusterName != "" {
		clusterPath := filepath.Join(base, "clusters", clusterName)
		candidates := []string{"kind-config.yaml", "kind-config.yml"}
		for _, name := range candidates {
			p := filepath.Join(clusterPath, name)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		// try broader glob for files starting with "kind"
		matches, _ := filepath.Glob(filepath.Join(clusterPath, "kind*.y*ml"))
		if len(matches) > 0 {
			return matches[0]
		}
	}

	// 2) look in CLI config directory root
	candidates := []string{"kind-config.yaml", "kind-config.yml"}
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
