package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

var rePRURL = regexp.MustCompile(`github\.com/([^/]+)/([^/]+)/pull/(\d+)`)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// PRInfo holds the metadata returned by the GitHub API for a pull request.
type PRInfo struct {
	Owner      string
	Repo       string
	Number     int
	BaseBranch string
	HeadBranch string
}

// FetchPRInfo resolves a GitHub PR URL to its base and head branch names.
// token may be empty for public repos; set GITHUB_TOKEN for private repos
// or to avoid rate limiting.
func FetchPRInfo(prURL, token string) (*PRInfo, error) {
	m := rePRURL.FindStringSubmatch(prURL)
	if m == nil {
		return nil, fmt.Errorf("not a valid GitHub PR URL (expected https://github.com/owner/repo/pull/NNN)")
	}
	owner, repo := m[1], m[2]
	number, _ := strconv.Atoi(m[3])

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d", owner, repo, number)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("PR not found: %s — check the URL and that the repo is public (or set GITHUB_TOKEN)", prURL)
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("GitHub API returned %d — set GITHUB_TOKEN for private repos or to avoid rate limiting", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d for %s", resp.StatusCode, apiURL)
	}

	var payload struct {
		Base struct{ Ref string `json:"ref"` } `json:"base"`
		Head struct{ Ref string `json:"ref"` } `json:"head"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub API response: %w", err)
	}
	if payload.Base.Ref == "" || payload.Head.Ref == "" {
		return nil, fmt.Errorf("GitHub API response missing base/head ref fields")
	}

	return &PRInfo{
		Owner:      owner,
		Repo:       repo,
		Number:     number,
		BaseBranch: payload.Base.Ref,
		HeadBranch: payload.Head.Ref,
	}, nil
}
