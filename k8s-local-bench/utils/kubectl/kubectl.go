package kubectl

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Client configures how kubectl is invoked. Optional fields may be nil.
type Client struct {
	// Path to kubectl binary. If nil, will resolve via PATH.
	KubectlPath *string
	// Kubeconfig file path. If nil, uses default kubeconfig behavior.
	Kubeconfig *string
	// Extra args to pass to kubectl (e.g., --namespace)
	ExtraArgs []string
}

// NewClient creates a basic Client.
func NewClient(Kubeconfig *string, ExtraArgs []string) *Client {
	return &Client{
		Kubeconfig: Kubeconfig,
		ExtraArgs:  ExtraArgs,
	}
}

// resolveKubectl returns the binary path to use.
func (c *Client) resolveKubectl() (string, error) {
	if c != nil && c.KubectlPath != nil && *c.KubectlPath != "" {
		return *c.KubectlPath, nil
	}
	p, err := exec.LookPath("kubectl")
	if err != nil {
		return "", fmt.Errorf("kubectl not found in PATH: %w", err)
	}
	return p, nil
}

// buildBaseArgs returns common kubectl args including kubeconfig if set.
func (c *Client) buildBaseArgs() []string {
	var args []string
	if c != nil && c.Kubeconfig != nil && *c.Kubeconfig != "" {
		args = append(args, "--kubeconfig", *c.Kubeconfig)
	}
	if c != nil && len(c.ExtraArgs) > 0 {
		args = append(args, c.ExtraArgs...)
	}
	return args
}

// ApplyPaths takes a list of glob patterns, expands them on the local filesystem,
// and runs `kubectl apply -f` with the matched files and directories. Patterns
// that don't match anything cause an error.
func (c *Client) ApplyPaths(ctx context.Context, patterns []string) error {
	if len(patterns) == 0 {
		return fmt.Errorf("no patterns provided")
	}

	var matches []string
	var unmatched []string
	for _, p := range patterns {
		// If the pattern looks like a URL, pass it through directly
		if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
			matches = append(matches, p)
			continue
		}

		// Expand ~ to home
		if strings.HasPrefix(p, "~") {
			if home, err := os.UserHomeDir(); err == nil {
				p = filepath.Join(home, strings.TrimPrefix(p, "~"))
			}
		}

		g, err := filepath.Glob(p)
		if err != nil {
			return fmt.Errorf("invalid glob pattern %q: %w", p, err)
		}
		if len(g) == 0 {
			unmatched = append(unmatched, p)
			continue
		}
		for _, m := range g {
			matches = append(matches, m)
		}
	}

	if len(unmatched) > 0 {
		return fmt.Errorf("patterns did not match any files: %v", unmatched)
	}

	if len(matches) == 0 {
		return fmt.Errorf("no files to apply")
	}

	kubectlPath, err := c.resolveKubectl()
	if err != nil {
		return err
	}

	// Build command: kubectl apply -f <item1> -f <item2> ... [--kubeconfig ...] [extra args]
	args := []string{"apply"}
	for _, m := range matches {
		args = append(args, "-f", m)
	}
	args = append(args, c.buildBaseArgs()...)

	cmd := exec.CommandContext(ctx, kubectlPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kubectl apply failed: %w", err)
	}
	return nil
}
