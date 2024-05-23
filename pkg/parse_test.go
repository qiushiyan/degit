package degit

import (
	"fmt"
	"testing"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		expected *Repo
		err      error
	}{
		{
			name: "GitHub repository with default branch",
			url:  "github.com/user/repo",
			expected: &Repo{
				Site: "github",
				User: "user",
				Name: "repo",
				Ref:  "HEAD",
				URL:  "https://github.com/user/repo",
			},
		},
		{
			name: "GitHub repository with commit hash",
			url:  "github.com/user/repo#759cf0f9d8f7b3828f7375f902742c7f4093d766",
			expected: &Repo{
				Site: "github",
				User: "user",
				Name: "repo",
				Ref:  "759cf0f9d8f7b3828f7375f902742c7f4093d766",
				URL:  "https://github.com/user/repo",
			},
		},
		{
			name: "GitLab repository with subdir and branch",
			url:  "https://gitlab.com/user/repo/subdir#branch",
			expected: &Repo{
				Site:   "gitlab",
				User:   "user",
				Name:   "repo",
				Ref:    "branch",
				URL:    "https://gitlab.com/user/repo",
				Subdir: "/subdir",
			},
		},

		{
			name: "Default to GitHub with branch",
			url:  "user/repo#branch",
			expected: &Repo{
				Site: "github",
				User: "user",
				Name: "repo",
				Ref:  "branch",
				URL:  "https://github.com/user/repo",
			},
		},

		{
			name: "Unsupported host",
			url:  "https://example.com/user/repo",
			err:  fmt.Errorf("degit supports GitHub, GitLab, Sourcehut and BitBucket"),
		},
		{
			name: "Invalid URL",
			url:  "invalid-url",
			err:  fmt.Errorf("could not parse invalid-url"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo, err := ParseRepo(tc.url)
			if tc.err != nil {
				if err == nil || err.Error() != tc.err.Error() {
					t.Errorf("Expected error '%v', got '%v'", tc.err, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !reposEqual(repo, tc.expected) {
					t.Errorf("Expected repo %+v, got %+v", tc.expected, repo)
				}
			}
		})
	}
}

func reposEqual(r1, r2 *Repo) bool {
	return r1.Site == r2.Site &&
		r1.User == r2.User &&
		r1.Name == r2.Name &&
		r1.Ref == r2.Ref &&
		r1.URL == r2.URL &&
		r1.Subdir == r2.Subdir
}
