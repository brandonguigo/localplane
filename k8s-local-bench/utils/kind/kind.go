package kind

import (
	"fmt"
	"k8s-local-bench/config"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/rs/zerolog/log"
)

// isInstalled returns true if the given executable is found in PATH.
func isInstalled(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// runCmd runs a command and returns combined stdout/stderr.
func runCmd(name string, args ...string) (string, error) {
	log.Debug().Str("cmd", name).Str("args", strings.Join(args, " ")).Msg("running command")
	out, err := exec.Command(name, args...).CombinedOutput()
	return string(out), err
}

// ensureCloudProviderKindInstalled ensures the `cloud-provider-kind` binary exists.
// If missing and `go` is available, it will attempt `go install sigs.k8s.io/cloud-provider-kind@latest`.
func ensureCloudProviderKindInstalled() error {
	if isInstalled("cloud-provider-kind") {
		return nil
	}
	if !isInstalled("go") {
		return fmt.Errorf("go not installed; cannot install cloud-provider-kind")
	}

	out, err := runCmd("go", "install", "sigs.k8s.io/cloud-provider-kind@latest")
	if err != nil {
		return fmt.Errorf("failed to install cloud-provider-kind: %w; output: %s", err, out)
	}
	if !isInstalled("cloud-provider-kind") {
		return fmt.Errorf("cloud-provider-kind not found in PATH after install; output: %s", out)
	}
	log.Info().Msg("cloud-provider-kind installed")
	return nil
}

// Client provides a small wrapper to configure common options for kind
// operations such as a default kubeconfig path.
type Client struct {
	Kubeconfig string
}

// NewClient creates a Client. Pass empty string for defaults.
func NewClient(kubeconfig string) *Client {
	return &Client{Kubeconfig: kubeconfig}
}

// Create creates a kind cluster with the provided name. If configPath is non-empty
// it will be passed to `kind create cluster --config`.
func (c *Client) Create(name string, configPath string) error {
	kubeconfigPath := c.Kubeconfig
	if !isInstalled("kind") {
		return fmt.Errorf("kind not installed")
	}
	if !isInstalled("docker") {
		return fmt.Errorf("docker not installed")
	}

	// ensure cloud-provider-kind is available (will attempt to install with `go install`)
	if err := ensureCloudProviderKindInstalled(); err != nil {
		return err
	}

	args := []string{"create", "cluster", "--name", name, "--kubeconfig", kubeconfigPath}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}

	out, err := runCmd("kind", args...)
	if err != nil {
		return fmt.Errorf("failed to create kind cluster: %w; output: %s", err, out)
	}
	log.Info().Str("name", name).Msg("kind cluster created")
	return nil
}

// Delete deletes a kind cluster by name.
func (c *Client) Delete(name string) error {
	if !isInstalled("kind") {
		return fmt.Errorf("kind not installed")
	}
	if !isInstalled("docker") {
		return fmt.Errorf("docker not installed")
	}

	out, err := runCmd("kind", "delete", "cluster", "--name", name)
	if err != nil {
		return fmt.Errorf("failed to delete kind cluster: %w; output: %s", err, out)
	}
	log.Info().Str("name", name).Msg("kind cluster deleted")
	return nil
}

// StartLoadBalancer starts the cloud-provider-kind process for the given cluster.
// If background==true the process is started detached and logs are written to
// a temp file; the function returns immediately while the process continues
// running after the CLI exits.
func (c *Client) StartLoadBalancer(clusterName string, background bool) error {
	clusterDirPath := filepath.Join(config.CliConfig.Directory, "clusters", clusterName, ".cloud-provider-kind")

	if err := ensureCloudProviderKindInstalled(); err != nil {
		return err
	}

	args := []string{}

	// determine whether we need sudo
	needSudo := os.Geteuid() != 0
	if needSudo && !isInstalled("sudo") {
		return fmt.Errorf("sudo required but not installed")
	}

	if !background {
		if needSudo {
			// run interactively so user can enter their sudo password
			cmd := exec.Command("sudo", append([]string{"cloud-provider-kind"}, args...)...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("cloud-provider-kind failed: %w", err)
			}
			return nil
		}
		out, err := runCmd("cloud-provider-kind", args...)
		if err != nil {
			return fmt.Errorf("cloud-provider-kind failed: %w; output: %s", err, out)
		}
		return nil
	}

	// background: if sudo is required, first validate sudo credentials interactively
	if needSudo {
		vcmd := exec.Command("sudo", "-v")
		vcmd.Stdout = os.Stdout
		vcmd.Stderr = os.Stderr
		vcmd.Stdin = os.Stdin
		if err := vcmd.Run(); err != nil {
			return fmt.Errorf("sudo validation failed: %w", err)
		}
	}

	// background: start detached with logs redirected to a log file next to pid
	if err := os.MkdirAll(clusterDirPath, 0o755); err != nil {
		return fmt.Errorf("failed to create cluster directory for logs/pid: %w", err)
	}
	logPath := filepath.Join(clusterDirPath, ".log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	var cmd *exec.Cmd
	if needSudo {
		cmd = exec.Command("sudo", append([]string{"cloud-provider-kind"}, args...)...)
	} else {
		cmd = exec.Command("cloud-provider-kind", args...)
	}
	cmd.Stdout = f
	cmd.Stderr = f
	cmd.Stdin = nil
	// detach from parent process (Unix)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		f.Close()
		return fmt.Errorf("failed to start cloud-provider-kind: %w", err)
	}

	// we intentionally do not wait; process should keep running after exit
	// write pid file so callers (delete) can find and kill the background process later
	pidPath := filepath.Join(clusterDirPath, ".pid")
	pidContent := fmt.Sprintf("%d\n", cmd.Process.Pid)
	if err := os.WriteFile(pidPath, []byte(pidContent), 0o644); err != nil {
		// log the error but continue; background process is running
		log.Error().Err(err).Str("path", pidPath).Msg("failed to write pid file")
	} else {
		log.Info().Str("pid_file", pidPath).Msg("wrote cloud-provider-kind pid file")
	}

	log.Info().Str("log", logPath).Int("pid", cmd.Process.Pid).Msg("cloud-provider-kind started in background")
	// close our file handle; child keeps file descriptor
	_ = f.Close()
	return nil
}

// StopLoadBalancer stops a previously-started background cloud-provider-kind
// process by reading the pid file, attempting to kill the process (using
// sudo if necessary), and removing the `.cloud-provider-kind` directory.
func (c *Client) StopLoadBalancer(clusterName string) error {
	clusterDirPath := filepath.Join(config.CliConfig.Directory, "clusters", clusterName, ".cloud-provider-kind")
	pidPath := filepath.Join(clusterDirPath, ".pid")

	data, err := os.ReadFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No pid file; try to remove directory anyway
			if err := os.RemoveAll(clusterDirPath); err != nil {
				return fmt.Errorf("failed to remove cloud-provider-kind directory: %w", err)
			}
			log.Info().Str("clusterDir", clusterDirPath).Msg("removed cloud-provider-kind directory (no pid file)")
			return nil
		}
		return fmt.Errorf("failed to read pid file: %w", err)
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return fmt.Errorf("invalid pid in file %s: %w", pidPath, err)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		log.Error().Err(err).Int("pid", pid).Msg("failed to find process")
	} else {
		// Try graceful termination first
		if err := proc.Signal(syscall.SIGTERM); err != nil {
			log.Warn().Err(err).Int("pid", pid).Msg("failed to send SIGTERM to process; attempting alternatives")
			// Try using sudo kill if available (process may be owned by root)
			if isInstalled("sudo") {
				out, e := runCmd("sudo", "kill", "-TERM", pidStr)
				if e != nil {
					log.Error().Err(e).Str("output", out).Msg("sudo kill -TERM failed; trying sudo kill -KILL")
					out2, e2 := runCmd("sudo", "kill", "-KILL", pidStr)
					if e2 != nil {
						log.Error().Err(e2).Str("output", out2).Msg("sudo kill -KILL failed")
						return fmt.Errorf("failed to kill process %d: %w", pid, e2)
					}
				}
			} else {
				// Fall back to os.Kill
				if e := proc.Kill(); e != nil {
					return fmt.Errorf("failed to kill process %d: %w", pid, e)
				}
			}
		}
	}

	// remove the cluster-specific cloud-provider-kind directory
	if err := os.RemoveAll(clusterDirPath); err != nil {
		return fmt.Errorf("failed to remove cloud-provider-kind directory: %w", err)
	}
	log.Info().Str("clusterDir", clusterDirPath).Msg("stopped cloud-provider-kind and removed directory")
	return nil
}
