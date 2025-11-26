package destroy

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/rs/zerolog/log"
)

// selectClusterInteractive lists existing kind clusters and prompts the user
// to select one. Returns the selected cluster name or an empty string on
// non-recoverable errors (caller should decide how to proceed).
func selectClusterInteractive() (string, error) {
	out, err := exec.Command("kind", "get", "clusters").CombinedOutput()
	outStr := strings.TrimSpace(string(out))
	if err != nil || outStr == "" {
		return "", fmt.Errorf("no kind clusters found")
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
		return "", fmt.Errorf("no kind clusters found")
	}

	// Protect against cases where the command returns a single placeholder
	// line such as "no kind clusters found" — treat that as no clusters.
	if len(clusters) == 1 {
		only := strings.TrimSpace(clusters[0])
		lower := strings.ToLower(only)
		if only == "" || strings.Contains(lower, "no kind") || strings.Contains(lower, "no clusters") {
			return "", fmt.Errorf("no kind clusters found")
		}
	}

	// use promptui to let the user select a cluster
	prompt := promptui.Select{
		Label: "Select cluster to destroy",
		Items: clusters,
		Size:  len(clusters),
	}
	_, result, err := prompt.Run()
	if err != nil {
		log.Info().Err(err).Msg("selection aborted or failed")
		return "", fmt.Errorf("selection aborted: %w", err)
	}
	return strings.TrimSpace(result), nil
}
