package realfs

import (
	"dedup/fs"
	"os"
	"path/filepath"

	"golang.org/x/text/unicode/norm"
)

type FS struct {
	root string
}

func New(path string) *FS {
	return &FS{root: path}
}

func (fsys *FS) Root() string {
	return fsys.root
}

func (fsys *FS) Scan(events fs.Events) {
}

func (fsys *FS) Remove(path string) {}

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
