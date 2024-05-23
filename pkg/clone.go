package degit

// Clone parses input string as a remote repository and downloads its tarball into the destination directory
func Clone(src string, dst string, force bool, verbose bool) error {
	repo, err := ParseRepo(src)
	if err != nil {
		return err
	}

	return repo.Clone(dst, force, verbose)
}
