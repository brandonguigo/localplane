package destroy

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

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
		return "", fmt.Errorf("invalid selection")
	}
	return clusters[choice-1], nil
}
