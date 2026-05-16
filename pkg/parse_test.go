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
			err: fmt.Errorf(
				"can't recognize %s as a git repository, example: github.com/user/repo",
				"invalid-url",
			),
		},

		{
			name: "GitHub web URL tree (folder)",
			url:  "https://github.com/u/r/tree/main/a/b",
			expected: &Repo{
				Site:   "github",
				User:   "u",
				Name:   "r",
				Ref:    "main",
				URL:    "https://github.com/u/r",
				Subdir: "/a/b",
				IsFile: false,
			},
		},
		{
			name: "GitHub web URL blob (file)",
			url:  "https://github.com/u/r/blob/main/README.md",
			expected: &Repo{
				Site:   "github",
				User:   "u",
				Name:   "r",
				Ref:    "main",
				URL:    "https://github.com/u/r",
				Subdir: "/README.md",
				IsFile: true,
			},
		},
		{
			name: "GitHub web URL blob with tag and nested path",
			url:  "https://github.com/u/r/blob/v1.2.3/docs/api.md",
			expected: &Repo{
				Site:   "github",
				User:   "u",
				Name:   "r",
				Ref:    "v1.2.3",
				URL:    "https://github.com/u/r",
				Subdir: "/docs/api.md",
				IsFile: true,
			},
		},
		{
			name: "GitHub web URL tree with commit SHA",
			url:  "https://github.com/u/r/tree/abc1234deadbeef/sub",
			expected: &Repo{
				Site:   "github",
				User:   "u",
				Name:   "r",
				Ref:    "abc1234deadbeef",
				URL:    "https://github.com/u/r",
				Subdir: "/sub",
				IsFile: false,
			},
		},
		{
			name: "GitHub raw URL",
			url:  "https://raw.githubusercontent.com/u/r/main/file.txt",
			expected: &Repo{
				Site:   "github",
				User:   "u",
				Name:   "r",
				Ref:    "main",
				URL:    "https://github.com/u/r",
				Subdir: "/file.txt",
				IsFile: true,
			},
		},
		{
			name: "GitHub HTTPS bare repo URL (regression)",
			url:  "https://github.com/u/r",
			expected: &Repo{
				Site:   "github",
				User:   "u",
				Name:   "r",
				Ref:    "HEAD",
				URL:    "https://github.com/u/r",
				Subdir: "",
				IsFile: false,
			},
		},
		{
			name: "Native syntax with fragment ref preserved (regression)",
			url:  "u/r/sub#main",
			expected: &Repo{
				Site:   "github",
				User:   "u",
				Name:   "r",
				Ref:    "main",
				URL:    "https://github.com/u/r",
				Subdir: "/sub",
				IsFile: false,
			},
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
		r1.Subdir == r2.Subdir &&
		r1.IsFile == r2.IsFile
}
