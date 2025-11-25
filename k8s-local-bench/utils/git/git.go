package git

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Client provides a simple API around common git operations rooted at Path.
type Client struct {
	Path string
}

// NewClient creates a git client that operates on the provided path.
func NewClient(path string) *Client {
	return &Client{Path: path}
}

// InitializeGitRepo ensures the client's Path is an empty directory (creates it if missing)
// and initializes an empty git repository there. It does not add any remotes/origins.
func (c *Client) InitializeGitRepo() error {
	path := c.Path
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("path %s exists and is not a directory", path)
		}
		// check directory is empty
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening directory %s: %w", path, err)
		}
		defer f.Close()
		// Try to read a single entry; if EOF -> empty
		_, err = f.Readdirnames(1)
		if err != nil && err != io.EOF {
			return fmt.Errorf("reading directory %s: %w", path, err)
		}
		if err != io.EOF {
			return fmt.Errorf("directory %s is not empty", path)
		}
	} else if os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", path, err)
		}
	} else {
		return fmt.Errorf("stat %s: %w", path, err)
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = path
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git init failed: %v: %s", err, string(out))
	}

	return nil
}

// CommitAll stages all changes under the client's Path and creates a commit with the
// provided message. The path must be a git repository working tree.
func (c *Client) CommitAll(message string) error {
	path := c.Path
	// git add --all
	cmd := exec.Command("git", "add", "--all")
	cmd.Dir = path
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git add failed: %v: %s", err, string(out))
	}

	// git commit -m <message>
	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = path
	out, err = cmd.CombinedOutput()
	if err != nil {
		// If there's nothing to commit, git returns exit code 1 with message on stdout/stderr.
		return fmt.Errorf("git commit failed: %v: %s", err, string(out))
	}

	return nil
}
