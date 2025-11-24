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
	argocdsvc "k8s-local-bench/utils/argocd"
	gitutil "k8s-local-bench/utils/git"
	"k8s-local-bench/utils/github"
	kindsvc "k8s-local-bench/utils/kind"
	kindcfg "k8s-local-bench/utils/kind/config"
	"k8s-local-bench/utils/kubectl"

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
	return cmd
}

func createCluster(cmd *cobra.Command, args []string) {
	log.Info().Msg("Creating local k8s cluster...")
	// honor CLI debug config if set
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
	// check for kind config file (looks inside CLI config directory clusters/<cluster-name>)
	kindCfgPath := findKindConfig(clusterName)
	var kindCfg *kindcfg.KindCluster

	if kindCfgPath == "" {
		log.Info().Msg("no kind config file found in current directory; creating default kind config")

		// create default kind config under CLI config directory: clusters/<name>/kind-config.yaml
		base := config.CliConfig.Directory
		var err error
		if base == "" {
			base, err = os.Getwd()
			if err != nil {
				log.Error().Err(err).Msg("failed to determine working directory for default kind config")
			}
		}
		clusterDir := filepath.Join(base, "clusters", clusterName)
		if err := os.MkdirAll(clusterDir, 0o755); err != nil {
			log.Error().Err(err).Str("path", clusterDir).Msg("failed to create cluster config directory")
		} else {
			defaultPath := filepath.Join(clusterDir, "kind-config.yaml")
			// basic default: single control-plane node
			def := &kindcfg.KindCluster{
				Kind:       "Cluster",
				APIVersion: "kind.x-k8s.io/v1alpha4",
				Nodes: []kindcfg.KindNode{{
					Role: "control-plane",
				}},
			}
			if err := kindcfg.SaveKindConfig(defaultPath, def); err != nil {
				log.Error().Err(err).Str("path", defaultPath).Msg("failed to write default kind config")
			} else {
				kindCfgPath = defaultPath
				kindCfg = def
				log.Info().Str("path", kindCfgPath).Msg("wrote default kind config")
			}
		}
	} else {
		log.Info().Str("path", kindCfgPath).Msg("found kind config file in current directory")

		// load and parse the kind config using the provided kindconfig struct
		if cfg, err := kindcfg.LoadKindConfig(kindCfgPath); err != nil {
			log.Error().Err(err).Str("path", kindCfgPath).Msg("failed to load kind config")
		} else {
			kindCfg = cfg
			log.Info().Str("kind", kindCfg.Kind).Str("apiVersion", kindCfg.APIVersion).Int("nodes", len(kindCfg.Nodes)).Msg("loaded kind config")
			for i, n := range cfg.Nodes {
				log.Debug().Int("nodeIndex", i).Str("role", n.Role).Int("extraMounts", len(n.ExtraMounts)).Msg("node details")
				for j, m := range n.ExtraMounts {
					log.Debug().Int("nodeIndex", i).Int("mountIndex", j).Str("hostPath", m.HostPath).Str("containerPath", m.ContainerPath).Msg("mount")
				}
			}
		}
	}

	// create local repository for ArgoCD to use (skip if disabled)
	if !disableArgoCD {
		base := config.CliConfig.Directory
		var err error
		if base == "" {
			base, err = os.Getwd()
			if err != nil {
				log.Error().Err(err).Msg("failed to determine working directory for local-argo repo")
				base = ""
			}
		}
		if base != "" {
			repoPath := filepath.Join(base, "local-argo")
			//create the repo path direcctory
			log.Debug().Str("path", repoPath).Msg("initializing local-argo git repo")
			if err := os.MkdirAll(repoPath, 0o755); err != nil {
				log.Fatal().Err(err).Str("path", repoPath).Msg("failed to create local-argo git repo directory")
			}
			if err := gitutil.InitializeGitRepo(repoPath); err != nil {
				log.Fatal().Err(err).Str("path", repoPath).Msg("failed to create local-argo git repo")
			} else {
				log.Info().Str("path", repoPath).Msg("created local-argo git repo")
			}
		} else {
			log.Debug().Msg("skipping local-argo repo creation; no base config directory available")
		}

		// update kindConfig to include the mount of the ArgoCD local repo
		if base != "" && kindCfgPath != "" {
			hostPath := filepath.Join(base, "local-argo")
			containerPath := "/mnt/local-argo"
			// ensure kindCfg is loaded
			if kindCfg == nil {
				if cfg, err := kindcfg.LoadKindConfig(kindCfgPath); err != nil {
					log.Debug().Err(err).Str("path", kindCfgPath).Msg("failed to reload kind config before adding mount")
				} else {
					kindCfg = cfg
				}
			}
			if kindCfg != nil {
				kindcfg.AddExtraMount(kindCfg, hostPath, containerPath)
				if err := kindcfg.SaveKindConfig(kindCfgPath, kindCfg); err != nil {
					log.Error().Err(err).Str("path", kindCfgPath).Msg("failed to write updated kind config with local-argo mount")
				} else {
					log.Info().Str("hostPath", hostPath).Str("containerPath", containerPath).Msg("added local-argo mount to kind config")
				}
			} else {
				log.Debug().Msg("no kind config available to patch with local-argo mount")
			}
		}

		// download local-stack directory from GitHub if not present into ArgoCD directory
		localStackHelmChartOwner := "brandonguigo"
		localStackHelmChartRepo := "k8s-local-bench"
		localStackHelmChartRef := "main"
		localStackHelmChartTemplatePath := "charts/local-stack"
		localStackPath := filepath.Join(base, "local-argo", "charts", "local-stack")
		log.Debug().Str("path", localStackPath).Msgf("checking for local-stack helm chart in local-argo repo")
		if _, err := os.Stat(localStackPath); os.IsNotExist(err) {
			log.Info().Str("path", localStackPath).Msgf("local-stack helm chart not found; downloading from GitHub repo %s/%s (ref: %s, path: %s)", localStackHelmChartOwner, localStackHelmChartRepo, localStackHelmChartRef, localStackHelmChartTemplatePath)
			err := github.DownloadRepoPath(cmd.Context(), localStackHelmChartOwner, localStackHelmChartRepo, localStackHelmChartRef, localStackHelmChartTemplatePath, localStackPath, "")
			if err != nil {
				log.Fatal().Err(err).Str("path", localStackPath).Msg("failed to download local-stack helm chart from GitHub")
			} else {
				log.Info().Str("path", localStackPath).Msg("downloaded local-stack helm chart from GitHub into local-argo repo")
			}

			// commit local-argo repo changes
			if base != "" {
				repoPath := filepath.Join(base, "local-argo")
				if err := gitutil.CommitAll(repoPath, "Update local-argo repo with local-stack helm chart"); err != nil {
					log.Error().Err(err).Str("path", repoPath).Msg("failed to commit changes to local-argo git repo")
				} else {
					log.Info().Str("path", repoPath).Msg("committed changes to local-argo git repo")
				}
			}
		} else {
			log.Info().Str("path", localStackPath).Msg("local-stack helm chart already exists; skipping download")
		}

	} else {
		log.Info().Msg("Argocd setup disabled; skipping ArgoCD related tasks")
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

	// deploy ArgoCD via Helm chart into the cluster with this config
	if !disableArgoCD {
		// build mounts for ArgoCD repoServer from CLI config directory
		mounts := []argocdsvc.RepoMount{{
			Name:      "local-argo",
			HostPath:  "/mnt/local-argo",
			MountPath: "/mnt/local-argo",
		}}

		// pass the current cluster kubeconfig path to ArgoCD install
		out, err := argocdsvc.InstallOrUpgradeArgoCD(mounts, kubeconfigPath)
		if err != nil {
			log.Fatal().Err(err).Str("output", out).Msg("failed to install argocd via helm sdk")
		} else {
			log.Info().Str("output", out).Msg("argocd installed")
		}
	}

	// install directory/local-argo/bootstrap/argo-bootstrap-*.yaml into the cluster (bootstrap argo repo and apps)
	kubectlClient := kubectl.NewClient(&kubeconfigPath, nil)
	base := config.CliConfig.Directory
	bootstrapPath := filepath.Join(base, "local-argo", "charts", "local-stack", "bootstrap")
	patterns := []string{filepath.Join(bootstrapPath, "argo-bootstrap-*.yaml")}
	log.Info().Strs("patterns", patterns).Msg("applying bootstrap manifests into cluster")
	if err := kubectlClient.ApplyPaths(cmd.Context(), patterns); err != nil {
		log.Fatal().Err(err).Msg("failed to apply bootstrap manifests into cluster")
	} else {
		log.Info().Msg("applied bootstrap manifests into cluster")
	}

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
