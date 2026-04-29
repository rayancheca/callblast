package github

import (
	"testing"
)

func TestFetchPRInfo_InvalidURL(t *testing.T) {
	cases := []string{
		"",
		"not-a-url",
		"https://gitlab.com/owner/repo/merge_requests/1",
		"https://github.com/owner/repo",
		"https://github.com/owner/repo/issues/1",
	}
	for _, url := range cases {
		_, err := FetchPRInfo(url, "")
		if err == nil {
			t.Errorf("expected error for URL %q, got nil", url)
		}
	}
}

func TestRePRURL_Matches(t *testing.T) {
	cases := []struct {
		url    string
		owner  string
		repo   string
		number string
	}{
		{"https://github.com/golang/go/pull/12345", "golang", "go", "12345"},
		{"https://github.com/anthropics/anthropic-sdk-go/pull/42", "anthropics", "anthropic-sdk-go", "42"},
		{"http://github.com/foo/bar/pull/1", "foo", "bar", "1"},
	}
	for _, tc := range cases {
		m := rePRURL.FindStringSubmatch(tc.url)
		if m == nil {
			t.Errorf("expected match for %q", tc.url)
			continue
		}
		if m[1] != tc.owner || m[2] != tc.repo || m[3] != tc.number {
			t.Errorf("URL %q: got owner=%s repo=%s number=%s, want %s/%s/%s",
				tc.url, m[1], m[2], m[3], tc.owner, tc.repo, tc.number)
		}
	}
}
