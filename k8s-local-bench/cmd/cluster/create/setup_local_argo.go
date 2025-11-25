package create

import (
	"os"
	"path/filepath"

	"k8s-local-bench/config"
	gitutil "k8s-local-bench/utils/git"
	"k8s-local-bench/utils/github"
	kindcfg "k8s-local-bench/utils/kind/config"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// setupLocalArgo performs creation of the local-argo git repo, patches the
// kind config with a mount, and downloads the local-stack chart if missing.
// It returns the resolved base directory, possibly-updated kindCfgPath and kindCfg.
func setupLocalArgo(cmd *cobra.Command, disableArgoCD bool, kindCfgPath string, kindCfg *kindcfg.KindCluster) (string, string, *kindcfg.KindCluster) {
	base := config.CliConfig.Directory
	var err error
	if base == "" {
		base, err = os.Getwd()
		if err != nil {
			log.Error().Err(err).Msg("failed to determine working directory for local-argo repo")
			base = ""
		}
	}

	if !disableArgoCD {
		if base != "" {
			repoPath := filepath.Join(base, "local-argo")
			log.Debug().Str("path", repoPath).Msg("initializing local-argo git repo")
			if err := os.MkdirAll(repoPath, 0o755); err != nil {
				log.Error().Err(err).Str("path", repoPath).Msg("failed to create local-argo git repo directory")
			}
			gitClient := gitutil.NewClient(repoPath)
			if err := gitClient.InitializeGitRepo(); err != nil {
				log.Error().Err(err).Str("path", repoPath).Msg("failed to create local-argo git repo")
			} else {
				log.Info().Str("path", repoPath).Msg("created local-argo git repo")
			}
		} else {
			log.Debug().Msg("skipping local-argo repo creation; no base config directory available")
		}

		// add mount to kind config for local-argo if available
		if base != "" && kindCfgPath != "" {
			hostPath := filepath.Join(base, "local-argo")
			containerPath := "/mnt/local-argo"
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

		// download local-stack helm chart into local-argo if missing
		localStackHelmChartOwner := "brandonguigo"
		localStackHelmChartRepo := "k8s-local-bench"
		localStackHelmChartRef := "main"
		localStackHelmChartTemplatePath := "charts/local-stack"
		localStackPath := filepath.Join(base, "local-argo", "charts", "local-stack")
		log.Debug().Str("path", localStackPath).Msg("checking for local-stack helm chart in local-argo repo")
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
				gitClient := gitutil.NewClient(repoPath)
				if err := gitClient.CommitAll("Update local-argo repo with local-stack helm chart"); err != nil {
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

	return base, kindCfgPath, kindCfg
}
