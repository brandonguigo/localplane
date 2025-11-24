package create

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"k8s-local-bench/config"
	kindsvc "k8s-local-bench/utils/kind"
	kindcfg "k8s-local-bench/utils/kind/config"

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
	// check for kind config file (looks inside CLI config directory clusters/<cluster-name>)
	kindCfgPath := findKindConfig(clusterName)
	var kindCfg kindcfg.KindCluster

	if kindCfgPath == "" {
		log.Info().Msg("no kind config file found in current directory; proceeding without one")
	} else {
		log.Info().Str("path", kindCfgPath).Msg("found kind config file in current directory")

		// load and parse the kind config using the provided kindconfig struct
		if cfg, err := kindcfg.LoadKindConfig(kindCfgPath); err != nil {
			log.Error().Err(err).Str("path", kindCfgPath).Msg("failed to load kind config")
		} else {
			kindCfg = *cfg
			log.Info().Str("kind", kindCfg.Kind).Str("apiVersion", kindCfg.APIVersion).Int("nodes", len(kindCfg.Nodes)).Msg("loaded kind config")
			for i, n := range cfg.Nodes {
				log.Debug().Int("nodeIndex", i).Str("role", n.Role).Int("extraMounts", len(n.ExtraMounts)).Msg("node details")
				for j, m := range n.ExtraMounts {
					log.Debug().Int("nodeIndex", i).Int("mountIndex", j).Str("hostPath", m.HostPath).Str("containerPath", m.ContainerPath).Msg("mount")
				}
			}
		}
	}

	// TODO: create kindConfig if not found with default settings

	// TODO: update kindConfig to include the mount of the ArgoCD local repo

	// TODO: create the argocd directory that will contain the local-stack chart

	// TODO: download local-stack template from GitHub if not present into ArgoCD directory

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
	// kubeconfig path will be in CLI config directory under clusters/<name>/kubeconfig
	kubeconfigPath := filepath.Join(config.CliConfig.Directory, "clusters", clusterName, "kubeconfig")
	if err := kindsvc.Create(clusterName, kindCfgPath, kubeconfigPath); err != nil {
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

	// verify cluster readiness: wait up to 3 minutes for pods to be healthy
	{
		timeout := 3 * time.Minute
		deadline := time.Now().Add(timeout)
		ctxName := "kind-" + clusterName
		for {
			out, err := exec.Command("kubectl", "--context", ctxName, "get", "pods", "--all-namespaces", "--no-headers").CombinedOutput()
			outStr := strings.TrimSpace(string(out))
			if err != nil {
				log.Debug().Err(err).Str("output", outStr).Msg("kubectl get pods failed; cluster may not be ready yet")
			}

			healthy := true
			if outStr == "" {
				healthy = false
			} else {
				lines := strings.Split(outStr, "\n")
				for _, l := range lines {
					f := strings.Fields(l)
					if len(f) < 4 {
						continue
					}
					status := f[3]
					if status == "Pending" || strings.Contains(status, "CrashLoopBackOff") || status == "Error" || status == "Failed" {
						healthy = false
						break
					}
				}
			}

			if healthy {
				log.Info().Str("context", ctxName).Msg("cluster pods are healthy")
				break
			}
			if time.Now().After(deadline) {
				log.Error().Str("context", ctxName).Msg("cluster did not become healthy within timeout")
				break
			}
			time.Sleep(5 * time.Second)
		}
	}

	// TODO: deploy ArgoCD via Helm chart into the cluster

	// TODO: create local repository for ArgoCD to use

	// TODO: install local-stack ArgoCD app into the cluster

	log.Info().Msg("local k8s cluster creation process completed")
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
