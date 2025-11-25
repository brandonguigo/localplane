package create

import (
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// waitForClusterReadiness polls kubectl to determine whether cluster pods are healthy.
func waitForClusterReadiness(clusterName string, timeout time.Duration) {
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
