package degit

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"time"
)

var accessLogName = "access.json"
var hashLogName = "map.json"

// ClearCache remove cache folder for repositories. If filter is empty, all caches are cleared.
func ClearCache(filter string, verbose bool) error {
	base := GetCacheDir()
	ok, err := exists(base)
	if err != nil {
		return err
	}
	if !ok {
		if verbose {
			fmt.Println("no cache found")
		}
		return nil
	}
	if filter == "" {
		return os.RemoveAll(base)
	}

	r, err := ParseRepo(filter)
	if err != nil {
		return err
	}

	dir := path.Join(base, r.Site, r.User, r.Name)
	ok, err = exists(dir)
	if err != nil {
		return err
	}
	if !ok {
		if verbose {
			fmt.Printf("no cache found for %s\n", filter)
		}
		return nil
	}

	return os.RemoveAll(dir)
}

// GetCacheDir returns the cache directory, usually $HOME/.go-degit
func GetCacheDir() string {
	return path.Join(homeOrTmp(), ".go-degit")
}

func updateCache(dir string, ref string, hash string, verbose bool) error {
	if err := updateAccessLog(dir, ref); err != nil {
		return err
	}

	if err := updateHashLog(dir, ref, hash, verbose); err != nil {
		return err
	}
	return nil
}

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

func updateHashLog(dir string, ref string, hash string, verbose bool) error {
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
			log(verbose, "removing outdated cache", oldFile)
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
