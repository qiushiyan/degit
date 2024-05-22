package degit

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var base = homeOrTmp()

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

// Clone copies the repository to the destination directory
func (r *Repo) Clone(dst string, force bool) error {
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
			return fmt.Errorf("output location %s already exists, use --force to overwrite", dst)
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
		fmt.Println("downloading", file, filepath.Dir(file))
		err := os.MkdirAll(filepath.Dir(file), os.ModePerm)
		if err != nil {
			return err
		}
		err = r.download(file, hash)
		if err != nil {
			return err
		}
	}

	if err := updateCache(filepath.Dir(file), r.Ref, hash); err != nil {
		return err
	}

	err = os.MkdirAll(dst, os.ModePerm)
	if err != nil {
		return err
	}

	return untar(file, dst, r.Subdir, fmt.Sprintf("%s-%s", r.Name, hash))
}

func (r *Repo) download(dst string, hash string) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	var url string
	switch r.Site {
	case "gitlab":
		url = fmt.Sprintf("%s/repository/archive.tar.gz?ref=%s", r.URL, hash)
	case "bitbucket":
		url = fmt.Sprintf("%s/get/%s.tar.gz", r.URL, hash)
	default:
		url = fmt.Sprintf("%s/archive/%s.tar.gz", r.URL, hash)
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("could not find repository %s", r.URL)
	}
	if resp.StatusCode != 200 {
		return r.download(dst, resp.Header.Values("Location")[0])
	}
	defer resp.Body.Close()

	_, err = io.Copy(f, resp.Body)

	return err
}

func (r *Repo) getOutputFile(hash string) string {
	return path.Join(base, ".go-degit", r.Site, r.User, r.Name, fmt.Sprintf("%s.tar.gz", hash))
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

	return "", fmt.Errorf("could not find ref %s", r.Ref)
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
		return nil, err
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
			re := regexp.MustCompile(`refs\/(\w+)\/(.+)`)
			match := re.FindStringSubmatch(r)
			if match == nil {
				return nil, errors.New(fmt.Sprintf("could not parse %s", r))
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

func updateCache(dir string, ref string, hash string) error {
	if err := updateAccessLog(dir, ref); err != nil {
		return err
	}

	if err := updateHashLog(dir, ref, hash); err != nil {
		return err
	}
	return nil
}

var accessLogName = "access.json"

func updateAccessLog(dir string, ref string) error {
	path := path.Join(dir, accessLogName)

	// Open the file with read-write permissions, create if it doesn't exist
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	var data map[string]any = make(map[string]any)

	// Read the existing contents if the file is not empty
	s, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	if len(s) != 0 {
		err = json.Unmarshal(s, &data)
		if err != nil {
			return err
		}
	}

	// Update the data with the new reference and timestamp
	data[ref] = time.Now()
	s, err = json.Marshal(data)
	if err != nil {
		return err
	}

	// Truncate the file to ensure it's empty before writing the new data
	if err := f.Truncate(0); err != nil {
		return err
	}

	// Move the file pointer to the beginning of the file
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}

	// Write the updated data
	_, err = f.Write(s)
	return err
}

var hashLogName = "map.json"

func updateHashLog(dir string, ref string, hash string) error {
	p := path.Join(dir, hashLogName)

	// Open the file with read-write permissions, create if it doesn't exist
	f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	var data = make(map[string]any)

	// Read the existing contents if the file is not empty
	s, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	if len(s) != 0 {
		err = json.Unmarshal(s, &data)
		if err != nil {
			return err
		}
	}

	// Check and remove the outdated cache file if the hash has changed
	if oldHash, ok := data[ref]; ok {
		if oldHash != hash {
			oldFile := path.Join(dir, fmt.Sprintf("%s.tar.gz", oldHash))
			os.Remove(oldFile)
			fmt.Println("Removing outdated cache", oldFile)
		}
	}
	data[ref] = hash

	s, err = json.Marshal(data)
	if err != nil {
		return err
	}

	// Truncate the file to ensure it's empty before writing the new data
	if err := f.Truncate(0); err != nil {
		return err
	}

	// Move the file pointer to the beginning of the file
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}

	// Write the updated data
	_, err = f.Write(s)
	return err
}

func homeOrTmp() string {
	if s, err := os.UserHomeDir(); s != "" && err == nil {
		return s
	}
	return os.TempDir()
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
