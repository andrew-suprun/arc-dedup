package realfs

import (
	"dedup/fs"
	"dedup/lifecycle"
	"os"
	"path/filepath"

	"golang.org/x/text/unicode/norm"
)

type FS struct {
	root string
	lc   *lifecycle.Lifecycle
}

func New(path string, lc *lifecycle.Lifecycle) *FS {
	return &FS{root: path, lc: lc}
}

func (fsys *FS) Root() string {
	return fsys.root
}

func (fsys *FS) Scan(events fs.Events) {
}

func (fsys *FS) Remove(path string, events fs.Events) {}

func AbsPath(path string) (string, error) {
	var err error
	path, err = filepath.Abs(path)
	path = norm.NFC.String(path)
	if err != nil {
		return "", err
	}

	_, err = os.Stat(path)
	if err != nil {
		return "", err
	}
	return path, nil
}
