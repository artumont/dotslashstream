package postgres

import (
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"
)

// readSQL reads a SQL file from the migration folder.
func readSQL(fsys fs.FS, dir, filename string) (string, error) {
	matches, err := fs.Glob(fsys, filepath.Join(dir, "*"+filename))
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("no %s in %s", filename, dir)
	}
	data, err := fs.ReadFile(fsys, matches[0])
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// gitAuthor reads the git config user.name, falling back to "unknown".
func gitAuthor() string {
	out, err := exec.Command("git", "config", "user.name").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}
