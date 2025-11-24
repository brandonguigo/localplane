package destroy

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"k8s-local-bench/config"
	kindsvc "k8s-local-bench/utils/kind"

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
	// if no cluster name provided, list existing kind clusters and ask user to pick one
	if strings.TrimSpace(clusterName) == "" {
		out, err := exec.Command("kind", "get", "clusters").CombinedOutput()
		outStr := strings.TrimSpace(string(out))
		if err != nil || outStr == "" {
			log.Info().Msg("no kind clusters found")
			return
		}
		lines := strings.Split(outStr, "\n")
		clusters := []string{}
		for _, l := range lines {
			l = strings.TrimSpace(l)
			if l != "" {
				clusters = append(clusters, l)
			}
		}
		if len(clusters) == 0 {
			log.Info().Msg("no kind clusters found")
			return
		}
		fmt.Println("Available kind clusters:")
		for i, c := range clusters {
			fmt.Printf("%d) %s\n", i+1, c)
		}
		fmt.Printf("Select cluster to destroy [1-%d]: ", len(clusters))
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > len(clusters) {
			log.Info().Msg("invalid selection; aborting")
			return
		}
		clusterName = clusters[choice-1]
	}
	// check for kind config file (looks inside CLI config directory clusters/<cluster-name>)
	kindCfg := findKindConfig(clusterName)
	if kindCfg == "" {
		log.Info().Msg("no kind config file found in current directory; proceeding without one")
	} else {
		log.Info().Str("path", kindCfg).Msg("found kind config file in current directory")
	}

	// delete kind cluster by name
	if err := kindsvc.Delete(clusterName); err != nil {
		log.Error().Err(err).Msg("failed deleting kind cluster")
	} else {
		log.Info().Str("name", clusterName).Msg("kind cluster deletion invoked")
	}

	// attempt to stop any running cloud-provider-kind process (best-effort)
	if out, err := exec.Command("pkill", "-f", "cloud-provider-kind").CombinedOutput(); err != nil {
		log.Debug().Err(err).Str("output", string(out)).Msg("pkill for cloud-provider-kind returned error (may be fine if not running)")
	} else {
		log.Info().Msg("stopped cloud-provider-kind processes")
	}

	// make sure the cluster is stopped/deleted: poll `kind get clusters` briefly
	const maxAttempts = 6
	deleteSuccess := false
	for i := 0; i < maxAttempts; i++ {
		out, err := exec.Command("kind", "get", "clusters").CombinedOutput()
		outStr := strings.TrimSpace(string(out))
		if err != nil {
			log.Debug().Err(err).Str("output", outStr).Msg("failed to list kind clusters")
		}
		if !strings.Contains(outStr, clusterName) {
			log.Info().Str("name", clusterName).Msg("cluster confirmed removed")
			deleteSuccess = true
			break
		}
		time.Sleep(1 * time.Second)
	}

	if !deleteSuccess {
		log.Warn().Str("name", clusterName).Msg("cluster still present after deletion attempts; manual cleanup may be needed")
		return
	}

	// delete the kubeconfig file under CLI config directory clusters/<name>/kubeconfig if cluster deletion was successful
	kubeconfigPath := filepath.Join(config.CliConfig.Directory, "clusters", clusterName, "kubeconfig")

	if err := os.Remove(kubeconfigPath); err != nil {
		log.Warn().Err(err).Str("path", kubeconfigPath).Msg("failed to delete kubeconfig file")
	} else {
		log.Info().Str("path", kubeconfigPath).Msg("deleted kubeconfig file")
	}
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
