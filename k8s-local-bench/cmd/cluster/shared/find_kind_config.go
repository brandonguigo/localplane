package shared

import (
	"os"
	"path/filepath"

	"k8s-local-bench/config"
)

// findKindConfig searches for common kind config filenames and returns the first found path or empty string.
func FindKindConfig(clusterName string) string {
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
