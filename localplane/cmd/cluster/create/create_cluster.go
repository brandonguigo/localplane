package create

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"localplane/cmd/cluster/shared"
	"localplane/config"

	kindsvc "localplane/utils/kind"
	kindcfg "localplane/utils/kind/config"
	"localplane/utils/kubectl"

	"github.com/briandowns/spinner"
	"github.com/manifoldco/promptui"
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
		prompt := promptui.Prompt{
			Label:   "Enter cluster name:",
			Default: "localplane",
		}

		input, err := prompt.Run()
		if err != nil {
			log.Debug().Err(err).Msg("prompt cancelled or failed; using default cluster name")
			clusterName = "localplane"
		} else {
			clusterName = strings.TrimSpace(input)
			if clusterName == "" {
				clusterName = "localplane"
			}
		}
	}

	// locate or create kind config
	log.Info().Str("cluster", clusterName).Msg("locating kind config")
	kindCfgPath := shared.FindKindConfig(clusterName)
	var kindCfg *kindcfg.KindCluster
	log.Debug().Str("path", kindCfgPath).Msg("kind config path located")

	log.Info().Str("path", kindCfgPath).Msg("loading or creating kind config")
	kindCfgPath, kindCfg = loadOrCreateKindConfig(kindCfgPath, clusterName)

	// ArgoCD / local-argo setup
	log.Info().Str("path", kindCfgPath).Msg("setting up ArgoCD inside the nodes")
	base, kindCfgPath, kindCfg := setupLocalArgo(cmd, disableArgoCD, kindCfgPath, kindCfg)
	log.Info().Str("path", kindCfgPath).Msg("kind config ready")

	// confirmation
	if !askCreateConfirmation(cmd, clusterName) {
		return
	}

	// create cluster
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Prefix = "Creating kind cluster... "
	s.Start()
	kubeconfigPath := filepath.Join(config.CliConfig.Directory, "clusters", clusterName, "kubeconfig")
	kindClient := kindsvc.NewClient(kubeconfigPath)
	if err := kindClient.Create(clusterName, kindCfgPath); err != nil {
		s.Stop()
		log.Error().Err(err).Msg("failed creating kind cluster")
		return
	}
	s.Stop()
	log.Info().Str("name", clusterName).Msg("kind cluster created")
	// start load balancer
	log.Info().Msg("starting local load balancer for LoadBalancer services")
	startLocalLoadBalancer(kindClient, cmd, clusterName)
	log.Info().Msg("local load balancer started")

	// wait for readiness
	s = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Prefix = "Waiting for cluster to be ready... "
	s.Start()
	waitForClusterReadiness(clusterName, 3*time.Minute)
	s.Stop()
	log.Info().Msg("cluster is ready")

	// install argocd if requested
	if disableArgoCD {
		log.Info().Msg("skipping ArgoCD installation as requested")
	} else {
		s = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Prefix = "Installing ArgoCD... "
		s.Start()
	}
	installArgoIfRequested(kubeconfigPath, disableArgoCD)
	s.Stop()
	if !disableArgoCD {
		log.Info().Msg("ArgoCD installed")
	}

	// apply bootstrap manifests
	if !disableArgoCD {
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Prefix = "Applying bootstrap manifests... "
		s.Start()
		applyBootstrapManifests(cmd, kubeconfigPath, base)
		s.Stop()
		log.Info().Msg("bootstrap manifests applied")
	} else {
		log.Info().Msg("skipping bootstrap manifests application as ArgoCD is disabled")
	}

	// wait for ingress to be ready inside the `ingress` namespace, then get the only
	// Service with type LoadBalancer (assumes the chart installs a single ingress
	// controller service of type LoadBalancer).
	ingressNs := "ingress"

	s = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Prefix = "Waiting for LoadBalancer service for ingress... "
	s.Start()
	svc, err := waitForLoadBalancerService(context.Background(), kubeconfigPath, ingressNs, 3*time.Minute, 5*time.Second)
	s.Stop()
	if err != nil {
		log.Warn().Err(err).Msg("did not find LoadBalancer service for ingress")
	} else {
		log.Info().Str("service", svc.Name).Str("namespace", svc.Namespace).Msg("found LoadBalancer service for ingress")
	}

	// update the dnsmasq configuration
	s = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Prefix = "Updating dnsmasq configuration... "
	s.Start()
	domain := "localplane"
	err = updateDnsmasqConfig(cmd, domain, svc.ExternalIPs[0])
	s.Stop()
	if err != nil {
		log.Error().Err(err).Msg("failed updating dnsmasq configuration")
	} else {
		log.Info().Str("domain", domain).Str("ip", svc.ExternalIPs[0]).Msg("updated dnsmasq configuration")
	}

	// display cluster infos
	argoCDUrl := "argocd" + "." + domain
	headlampUrl := "headlamp" + "." + domain
	kubectlClient := kubectl.NewClient(&kubeconfigPath, nil)
	headlampSecret, err := kubectlClient.CreateToken(context.TODO(), "headlamp", "monitoring")
	if err != nil {
		log.Error().Err(err).Msg("failed creating headlamp token")
	}

	displayClusterInfo(clusterName, kubeconfigPath, argoCDUrl, headlampUrl, headlampSecret)

	log.Info().Msg("local k8s cluster creation process completed")
}
