package destroy

import (
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// waitForClusterStopped polls `kind get clusters` up to maxAttempts times with
// the provided interval between attempts. Returns true if the cluster is no
// longer listed, false if still present after attempts.
func waitForClusterStopped(clusterName string, maxAttempts int, interval time.Duration) bool {
	for i := 0; i < maxAttempts; i++ {
		out, err := exec.Command("kind", "get", "clusters").CombinedOutput()
		outStr := strings.TrimSpace(string(out))
		if err != nil {
			log.Debug().Err(err).Str("output", outStr).Msg("failed to list kind clusters")
		}
		if !strings.Contains(outStr, clusterName) {
			log.Info().Str("name", clusterName).Msg("cluster confirmed removed")
			return true
		}
		time.Sleep(interval)
	}
	return false
}
