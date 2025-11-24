package argocd

import (
	"fmt"
	stdlog "log"
	"os"

	"github.com/rs/zerolog/log"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
)

// RepoMount defines a name/hostPath/mountPath triple for mounting a repository
// into Argo CD's repo-server.
type RepoMount struct {
	Name      string
	HostPath  string
	MountPath string
}

// InstallOrUpgradeArgoCD installs or upgrades ArgoCD using the Helm SDK (upgrade --install).
// - mounts: list of RepoMount to add to repoServer.volumes and repoServer.volumeMounts
// - kubeconfig: optional path to kubeconfig file (when non-empty the helm REST client will use it)
func InstallOrUpgradeArgoCD(mounts []RepoMount, kubeconfig string) (string, error) {
	// use official argo-cd chart from Argo Helm
	release := "argocd"
	namespace := "argocd"
	repoURL := "https://argoproj.github.io/argo-helm"
	chart := "argo/argo-cd"
	// prepare values map
	values := map[string]interface{}{}
	repoServer := map[string]interface{}{}
	server := map[string]interface{}{}

	vols := []interface{}{}
	vms := []interface{}{}

	global := map[string]interface{}{}
	config := map[string]interface{}{}

	for _, m := range mounts {
		vols = append(vols, map[string]interface{}{
			"name": m.Name,
			"hostPath": map[string]interface{}{
				"path": m.HostPath,
				"type": "Directory",
			},
		})
		vms = append(vms, map[string]interface{}{
			"name":      m.Name,
			"mountPath": m.MountPath,
		})
	}
	repoServer["volumes"] = vols
	repoServer["volumeMounts"] = vms

	// add an ingress with the local dnsmasq domain (argocd.k8s-bench.local)
	host := "argocd.k8s-bench.local"
	global["domain"] = host
	config["params"] = map[string]interface{}{
		"server.insecure": "true",
	}

	server["ingress"] = map[string]interface{}{
		"enabled": true,
		"tls":     false,
	}

	values["global"] = global
	values["config"] = config
	values["repoServer"] = repoServer
	values["server"] = server

	// helm SDK config
	settings := cli.New()
	if kubeconfig != "" {
		settings.KubeConfig = kubeconfig
	}
	var cfg action.Configuration
	if err := cfg.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), stdlog.Printf); err != nil {
		return "", fmt.Errorf("failed to init helm configuration: %w", err)
	}

	// locate and load chart (supports repo URL via ChartPathOptions)
	cp := action.ChartPathOptions{RepoURL: repoURL}
	chartPath, err := cp.LocateChart(chart, settings)
	if err != nil {
		return "", fmt.Errorf("locate chart: %w", err)
	}
	ch, err := loader.Load(chartPath)
	if err != nil {
		return "", fmt.Errorf("load chart: %w", err)
	}

	u := action.NewUpgrade(&cfg)
	u.Install = true
	u.Namespace = namespace
	u.Wait = true
	rel, err := u.Run(release, ch, values)
	if err != nil {
		return "", fmt.Errorf("upgrade/install failed: %w", err)
	}

	log.Info().Str("release", rel.Name).Int("version", rel.Version).Str("namespace", namespace).Msg("argocd installed/updated via helm sdk")
	return fmt.Sprintf("release %s (version %d)", rel.Name, rel.Version), nil
}
