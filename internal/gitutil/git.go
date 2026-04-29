package gitutil

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// ChangedFiles returns the list of files changed between base and head.
func ChangedFiles(repoPath, base, head string) ([]string, error) {
	out, err := gitCmd(repoPath, "diff", "--name-only", base+"..."+head)
	if err != nil {
		// Fallback: two-dot diff
		out, err = gitCmd(repoPath, "diff", "--name-only", base+".."+head)
		if err != nil {
			return nil, fmt.Errorf("git diff: %w", err)
		}
	}
	return parseLines(out), nil
}

// FileAtRef returns the content of a file at a specific git ref.
// Returns nil, nil if the file doesn't exist at that ref.
func FileAtRef(repoPath, ref, path string) ([]byte, error) {
	out, err := gitCmd(repoPath, "show", ref+":"+path)
	if err != nil {
		// File may not exist at this ref (new file)
		return nil, nil
	}
	return out, nil
}

// AllSourceFiles returns all tracked Go and TypeScript source files in the repo.
func AllSourceFiles(repoPath string) ([]string, error) {
	out, err := gitCmd(repoPath, "ls-files", "--cached", "--others", "--exclude-standard")
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	var result []string
	for _, f := range parseLines(out) {
		ext := strings.ToLower(filepath.Ext(f))
		if ext == ".go" || ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" {
			result = append(result, f)
		}
	}
	return result, nil
}

// RepoRoot returns the absolute path of the git repo root.
func RepoRoot(path string) (string, error) {
	out, err := gitCmd(path, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	lines := parseLines(out)
	if len(lines) == 0 {
		return "", fmt.Errorf("could not determine repo root")
	}
	return lines[0], nil
}

// HeadRef returns the current HEAD ref (branch name or commit hash).
func HeadRef(repoPath string) (string, error) {
	out, err := gitCmd(repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	lines := parseLines(out)
	if len(lines) == 0 {
		return "", fmt.Errorf("empty HEAD ref")
	}
	return lines[0], nil
}

func gitCmd(repoPath string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

func parseLines(data []byte) []string {
	raw := strings.TrimSpace(string(data))
	if raw == "" {
		return nil
	}
	return strings.Split(raw, "\n")
}
