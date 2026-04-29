package gitutil

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// validRef matches safe git ref names: branch names, SHAs, HEAD~N, HEAD^, etc.
var validRef = regexp.MustCompile(`^[a-zA-Z0-9._/\-~^@{}:]+$`)

// validateRef returns an error if the ref contains characters that could be
// used to inject unexpected arguments into git commands.
func validateRef(ref string) error {
	if ref == "" {
		return fmt.Errorf("ref must not be empty")
	}
	if !validRef.MatchString(ref) {
		return fmt.Errorf("ref contains invalid characters: %q", ref)
	}
	return nil
}

// ChangedFiles returns the list of files changed between base and head.
func ChangedFiles(repoPath, base, head string) ([]string, error) {
	if err := validateRef(base); err != nil {
		return nil, fmt.Errorf("invalid base ref: %w", err)
	}
	if err := validateRef(head); err != nil {
		return nil, fmt.Errorf("invalid head ref: %w", err)
	}
	out, err := gitCmd(repoPath, "diff", "--name-only", base+"..."+head)
	if err != nil {
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
	if err := validateRef(ref); err != nil {
		return nil, fmt.Errorf("invalid ref: %w", err)
	}
	// path must be relative and not escape the repo root
	clean := filepath.Clean(path)
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return nil, fmt.Errorf("path must be relative and within the repo: %q", path)
	}
	out, err := gitCmd(repoPath, "show", ref+":"+clean)
	if err != nil {
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
