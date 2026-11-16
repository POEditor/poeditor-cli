package transfer

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// WriteFile puts data at path unless the file already holds exactly that.
// The write is atomic: a temp file in the same directory, then a rename, so
// an interrupted pull never leaves half a file. With dryRun nothing is
// written and the status is WouldCreate or WouldUpdate.
func WriteFile(path string, data []byte, dryRun bool) (Status, error) {
	old, err := os.ReadFile(path)
	status := Updated
	switch {
	case errors.Is(err, fs.ErrNotExist):
		status = Created
	case err != nil:
		return "", err
	case bytes.Equal(old, data):
		return Unchanged, nil
	}
	if dryRun {
		return status.dryRun(), nil
	}

	mode := fs.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(dir, ".poeditor-*.tmp")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Chmod(tmp.Name(), mode); err != nil {
		return "", err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return "", err
	}
	return status, nil
}
