package mockfs

import (
	"cmp"
	"dedup/fs"
	"encoding/csv"
	"os"
	"slices"
	"strconv"
	"time"
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
	go fsys.scan(events)
}

func (fsys *FS) Remove(path string, events fs.Events) {}

func (fsys *FS) scan(events fs.Events) {
	time.Sleep(time.Second)
	metas := readMetas()
	for i := range metas {
		metas[i].Hash = ""
	}
	events.Send(metas)
	metas = readMetas()
	for _, file := range metas {
		events.Send(fs.FileHashed{
			Path: file.Path,
			Hash: file.Hash,
		})
		time.Sleep(time.Millisecond)
	}

	events.Send(fs.ArchiveHashed{})
}

func readMetas() fs.FileMetas {
	result := []fs.FileMeta{}
	hashInfoFile, err := os.Open("data/.meta.csv")
	if err != nil {
		return nil
	}
	defer hashInfoFile.Close()

	records, err := csv.NewReader(hashInfoFile).ReadAll()
	if err != nil || len(records) == 0 {
		return nil
	}

	for _, record := range records[1:] {
		if len(record) == 5 {
			name := record[1]
			size, er2 := strconv.ParseUint(record[2], 10, 64)
			modTime, er3 := time.Parse(time.RFC3339, record[3])
			modTime = modTime.UTC().Round(time.Second)
			hash := record[4]
			if hash == "" || er2 != nil || er3 != nil {
				continue
			}

			result = append(result, fs.FileMeta{
				Path:    name,
				Hash:    hash,
				Size:    int(size),
				ModTime: modTime,
			})
		}
	}
	slices.SortFunc(result, func(a, b fs.FileMeta) int {
		return cmp.Compare(a.Path, b.Path)
	})
	return result
}
