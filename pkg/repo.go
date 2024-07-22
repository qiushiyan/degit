package degit

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// Repo represents a remote repository at a ref (commit, branch, tag)
type Repo struct {
	Site   string
	User   string
	Name   string
	Ref    string
	URL    string
	SSH    string
	Subdir string
}

// Clone downloads the repository into the destination
func (r *Repo) Clone(dst string, force bool, verbose bool) error {
	dstExists, err := exists(dst)
	if err != nil {
		return err
	}
	if dstExists {
		if force {
			err = os.RemoveAll(dst)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("output location %s already exists", dst)
		}
	}

	refs, err := r.getRefs()
	if err != nil {
		return err
	}

	hash, err := r.getHash(refs)
	if err != nil {
		return err
	}

	file := r.getOutputFile(hash)
	fileExists, err := exists(file)
	if err != nil {
		return err
	}

	if !fileExists {
		err := os.MkdirAll(filepath.Dir(file), os.ModePerm)
		if err != nil {
			return err
		}
		err = r.download(file, hash, verbose)
		if err != nil {
			return err
		}
	} else {
		log(verbose, "using cache for", r.URL)
	}

	if err := updateCache(filepath.Dir(file), r.Ref, hash, verbose); err != nil {
		return err
	}

	err = os.MkdirAll(dst, os.ModePerm)
	if err != nil {
		return err
	}

	err = untar(file, dst, r.Subdir, fmt.Sprintf("%s-%s", r.Name, hash))
	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) download(dst string, hash string, verbose bool) error {
	folder, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer folder.Close()

	var url string
	switch r.Site {
	case "gitlab":
		url = fmt.Sprintf("%s/repository/archive.tar.gz?ref=%s", r.URL, hash)
	case "bitbucket":
		url = fmt.Sprintf("%s/get/%s.tar.gz", r.URL, hash)
	default:
		url = fmt.Sprintf("%s/archive/%s.tar.gz", r.URL, hash)
	}

	log(verbose, "downloading from", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("could not find repository %s", r.URL)
	}
	if resp.StatusCode != 200 {
		return r.download(dst, resp.Header.Values("Location")[0], verbose)
	}
	defer resp.Body.Close()

	_, err = io.Copy(folder, resp.Body)

	return err
}

func (r *Repo) getOutputFile(hash string) string {
	return path.Join(GetCacheDir(), r.Site, r.User, r.Name, fmt.Sprintf("%s.tar.gz", hash))
}

func (r *Repo) getHash(refs []*ref) (string, error) {

	if r.Ref == "HEAD" {
		for i := range refs {
			if refs[i].Type == "HEAD" {
				return refs[i].Hash, nil
			}
		}
	}

	// pick by branch or pr name
	for i := range refs {
		if refs[i].Name == r.Ref {
			return refs[i].Hash, nil
		}
	}

	// pick by commit hash
	if len(r.Ref) < 7 {
		return "", fmt.Errorf("commit hash %s is too short, must be at least 7 characters", r.Ref)
	}

	for i := range refs {
		if strings.HasPrefix(refs[i].Hash, r.Ref) {
			return refs[i].Hash, nil
		}
	}

	return "", fmt.Errorf("could not find ref %s for repo %s", r.Ref, r.URL)
}

func log(verbose bool, msg ...any) {
	if verbose {
		fmt.Println(msg...)
	}
}

type ref struct {
	Type string
	Name string
	Hash string
}

func (r *Repo) getRefs() ([]*ref, error) {
	cmd := exec.Command("git", "ls-remote", r.URL)

	var output []byte
	var err error

	if output, err = cmd.Output(); err != nil {
		return nil, fmt.Errorf("could not find repository %s", r.URL)
	}

	var result []*ref
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "\t")
		hash := fields[0]
		r := fields[1]
		if fields[1] == "HEAD" {
			result = append(result, &ref{
				Type: "HEAD",
				Hash: hash,
			})

		} else {
			re := regexp.MustCompile(`refs\/([\w-]+)\/(.+)`)
			match := re.FindStringSubmatch(r)
			if match == nil {
				return nil, fmt.Errorf("could not parse git history %s", r)
			}

			var refType string
			switch match[1] {
			case "heads":
				refType = "branch"
			case "refs":
				refType = "ref"
			default:
				refType = match[1]
			}

			result = append(result, &ref{
				Type: refType,
				Name: match[2],
				Hash: hash,
			})
		}

	}

	return result, err
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
