package degit

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var supportedHosts = map[string]bool{
	"github":    true,
	"gitlab":    true,
	"bitbucket": true,
	"sourcehut": true,
	"git.sr.ht": true,
}

func ParseRepo(src string) (*Repo, error) {
	if strings.HasPrefix(src, "https://") {
		if r, ok := tryParseWebURL(src); ok {
			return r, nil
		}
	}

	re := regexp.MustCompile(
		`^(?:(?:https:\/\/)?([^:/]+\.[^:/]+)\/|git@([^:/]+)[:/]|([^/]+):)?([^/\s]+)\/([^/\s#]+)(?:((?:\/[^/\s#]+)+))?(?:\/)?(?:#(.+))?`,
	)
	match := re.FindStringSubmatch(src)
	if match == nil {
		return nil, fmt.Errorf(
			"can't recognize %s as a git repository, example: github.com/user/repo",
			src,
		)
	}

	site := firstNonEmpty(match[1], match[2], match[3], "github")
	site = strings.TrimSuffix(site, ".com")
	site = strings.TrimSuffix(site, ".org")

	if _, ok := supportedHosts[site]; !ok {
		return nil, errors.New("degit supports GitHub, GitLab, Sourcehut and BitBucket")
	}

	user := match[4]
	name := strings.TrimSuffix(match[5], ".git")
	subdir := match[6]
	ref := match[7]
	if ref == "" {
		ref = "HEAD"
	}

	var domain string
	if site == "bitbucket" {
		domain = "bitbucket.org"
	} else if site == "git.sr.ht" {
		domain = "git.sr.ht"
	} else {
		domain = site + ".com"
	}

	repoURL := fmt.Sprintf("https://%s/%s/%s", domain, user, name)
	ssh := fmt.Sprintf("git@%s:%s/%s", domain, user, name)

	return &Repo{
		Site:   site,
		User:   user,
		Name:   name,
		Ref:    ref,
		URL:    repoURL,
		SSH:    ssh,
		Subdir: subdir,
	}, nil
}

// tryParseWebURL recognizes paste-from-browser HTTPS URLs for known web hosts.
// Returns (repo, true) on a clean structural match; (nil, false) signals the
// caller to fall through to the regex parser (which handles native syntax,
// SSH URLs, and the `#ref` fragment form).
func tryParseWebURL(src string) (*Repo, bool) {
	u, err := url.Parse(src)
	if err != nil || u.Fragment != "" {
		return nil, false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	switch u.Host {
	case "github.com":
		return parseGitHubWebURL(parts)
	case "raw.githubusercontent.com":
		return parseGitHubRawURL(parts)
	}
	return nil, false
}

func parseGitHubWebURL(parts []string) (*Repo, bool) {
	if len(parts) < 2 {
		return nil, false
	}
	user := parts[0]
	name := strings.TrimSuffix(parts[1], ".git")

	if len(parts) == 2 {
		return buildGitHubRepo(user, name, "HEAD", "", false), true
	}

	if len(parts) < 5 {
		return nil, false
	}
	kind, ref := parts[2], parts[3]
	path := "/" + strings.Join(parts[4:], "/")

	switch kind {
	case "tree":
		return buildGitHubRepo(user, name, ref, path, false), true
	case "blob":
		return buildGitHubRepo(user, name, ref, path, true), true
	}
	return nil, false
}

func parseGitHubRawURL(parts []string) (*Repo, bool) {
	if len(parts) < 4 {
		return nil, false
	}
	user := parts[0]
	name := strings.TrimSuffix(parts[1], ".git")
	ref := parts[2]
	path := "/" + strings.Join(parts[3:], "/")
	return buildGitHubRepo(user, name, ref, path, true), true
}

func buildGitHubRepo(user, name, ref, subdir string, isFile bool) *Repo {
	return &Repo{
		Site:   "github",
		User:   user,
		Name:   name,
		Ref:    ref,
		URL:    fmt.Sprintf("https://github.com/%s/%s", user, name),
		SSH:    fmt.Sprintf("git@github.com:%s/%s", user, name),
		Subdir: subdir,
		IsFile: isFile,
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
