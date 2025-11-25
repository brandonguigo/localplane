package create

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"k8s-local-bench/cmd/cluster/shared"
	"k8s-local-bench/config"

	kindsvc "k8s-local-bench/utils/kind"
	kindcfg "k8s-local-bench/utils/kind/config"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// createCluster is the main entrypoint invoked by the cobra command.
func createCluster(cmd *cobra.Command, args []string) {
	log.Info().Msg("Creating local k8s cluster...")
	if config.CliConfig.Debug {
		log.Debug().Bool("debug", true).Msg("debug enabled")
	}

	disableArgoCD, _ := cmd.Flags().GetBool("disable-argocd")

	// get cluster name and locate kind config inside CLI config clusters/<name>
	clusterName, _ := cmd.Flags().GetString("cluster-name")
	if strings.TrimSpace(clusterName) == "" {
		fmt.Printf("Enter cluster name (default 'local-bench'): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			clusterName = "local-bench"
		} else {
			clusterName = input
		}
	}

	// locate or create kind config
	kindCfgPath := shared.FindKindConfig(clusterName)
	var kindCfg *kindcfg.KindCluster
	kindCfgPath, kindCfg = loadOrCreateKindConfig(kindCfgPath, clusterName)

	// ArgoCD / local-argo setup
	base, kindCfgPath, kindCfg := setupLocalArgo(cmd, disableArgoCD, kindCfgPath, kindCfg)

	// confirmation
	if !askCreateConfirmation(cmd, clusterName) {
		return
	}

	// create cluster
	kubeconfigPath := filepath.Join(config.CliConfig.Directory, "clusters", clusterName, "kubeconfig")
	kindClient := kindsvc.NewClient(kubeconfigPath)
	if err := kindClient.Create(clusterName, kindCfgPath); err != nil {
		log.Error().Err(err).Msg("failed creating kind cluster")
		return
	}
	log.Info().Str("name", clusterName).Msg("kind cluster creation invoked")

	// start load balancer
	startLocalLoadBalancer(kindClient, cmd, clusterName)

	// wait for readiness
	waitForClusterReadiness(clusterName, 3*time.Minute)

	// install argocd if requested
	installArgoIfRequested(kubeconfigPath, disableArgoCD)

	// apply bootstrap manifests
	applyBootstrapManifests(cmd, kubeconfigPath, base)

	log.Info().Msg("local k8s cluster creation process completed")
}
