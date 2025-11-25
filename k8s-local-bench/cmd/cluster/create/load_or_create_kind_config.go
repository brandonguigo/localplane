package create

import (
    "os"
    "path/filepath"

    "k8s-local-bench/config"
    kindcfg "k8s-local-bench/utils/kind/config"

    "github.com/rs/zerolog/log"
)

// loadOrCreateKindConfig will either load an existing kind config at the
// provided path or create a basic default kind config under the CLI config
// directory clusters/<name>/kind-config.yaml when path is empty. It returns
// the resolved path and the parsed KindCluster (or nil on failure).
func loadOrCreateKindConfig(kindCfgPath, clusterName string) (string, *kindcfg.KindCluster) {
    if kindCfgPath == "" {
        log.Info().Msg("no kind config file found in current directory; creating default kind config")
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
            return "", nil
        }
        defaultPath := filepath.Join(clusterDir, "kind-config.yaml")
        def := &kindcfg.KindCluster{
            Kind:       "Cluster",
            APIVersion: "kind.x-k8s.io/v1alpha4",
            Nodes: []kindcfg.KindNode{{
                Role: "control-plane",
            }},
        }
        if err := kindcfg.SaveKindConfig(defaultPath, def); err != nil {
            log.Error().Err(err).Str("path", defaultPath).Msg("failed to write default kind config")
            return "", nil
        }
        log.Info().Str("path", defaultPath).Msg("wrote default kind config")
        return defaultPath, def
    }

    log.Info().Str("path", kindCfgPath).Msg("found kind config file in current directory")
    if cfg, err := kindcfg.LoadKindConfig(kindCfgPath); err != nil {
        log.Error().Err(err).Str("path", kindCfgPath).Msg("failed to load kind config")
        return kindCfgPath, nil
    } else {
        log.Info().Str("kind", cfg.Kind).Str("apiVersion", cfg.APIVersion).Int("nodes", len(cfg.Nodes)).Msg("loaded kind config")
        for i, n := range cfg.Nodes {
            log.Debug().Int("nodeIndex", i).Str("role", n.Role).Int("extraMounts", len(n.ExtraMounts)).Msg("node details")
            for j, m := range n.ExtraMounts {
                log.Debug().Int("nodeIndex", i).Int("mountIndex", j).Str("hostPath", m.HostPath).Str("containerPath", m.ContainerPath).Msg("mount")
            }
        }
        return kindCfgPath, cfg
    }
}
