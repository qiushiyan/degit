package degit

import (
	"errors"
	"fmt"
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

func Parse(src string) (*Repo, error) {
	re := regexp.MustCompile(
		`^(?:(?:https:\/\/)?([^:/]+\.[^:/]+)\/|git@([^:/]+)[:/]|([^/]+):)?([^/\s]+)\/([^/\s#]+)(?:((?:\/[^/\s#]+)+))?(?:\/)?(?:#(.+))?`,
	)
	match := re.FindStringSubmatch(src)
	if match == nil {
		return nil, errors.New(fmt.Sprintf("could not parse %s", src))
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

	url := fmt.Sprintf("https://%s/%s/%s", domain, user, name)
	ssh := fmt.Sprintf("git@%s:%s/%s", domain, user, name)

	return &Repo{
		Site:   site,
		User:   user,
		Name:   name,
		Ref:    ref,
		URL:    url,
		SSH:    ssh,
		Subdir: subdir,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
