package app

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"dedup/fs"
)

type (
	app struct {
		fs          fs.FS
		rootFolder  *file
		curFolder   *file
		byHash      map[string][]*file
		nDuplicates int
		hashing     int
		hashed      int
		state       appState

		folderTargets []folderTarget
		sortTargets   []sortTarget
		screenWidth   int
		screenHeight  int
		lastClickTime time.Time

		events events
	}

	file struct {
		name    string
		size    int
		modTime time.Time
		hash    string
		parent  *file
		dups    int
		*folder
	}

	files []*file

	folder struct {
		children      files
		selectedIdx   int
		offsetIdx     int
		sortColumn    sortColumn
		sortAscending []bool
		sorted        bool
	}

	appState int

	sortColumn int

	folderTarget struct {
		path   []string
		offset int
		width  int
	}

	sortTarget struct {
		sortColumn
		offset int
		width  int
	}
)

const (
	archiveScanning appState = iota
	archiveHashing
	archiveReady
)

const (
	sortByName sortColumn = iota
	sortByTime
	sortBySize
)

func (f *file) String() string {
	if f == nil {
		return "<nil>"
	}
	buf := &strings.Builder{}
	if f.folder == nil {
		buf.WriteString("file")
	} else {
		buf.WriteString("folder")
	}
	fmt.Fprintf(buf, "{name: %q, path: %v, size: %d, modTime: %s", f.name, f.path(), f.size, f.modTime.Format(time.DateTime))
	if f.hash != "" {
		fmt.Fprintf(buf, ", hash: %q", f.hash)
	}
	if f.dups > 0 {
		fmt.Fprintf(buf, ", dups: %d", f.dups)
	}
	buf.WriteRune('}')
	return buf.String()
}

func (f *file) findChild(name string) *file {
	if f.folder == nil {
		return nil
	}
	for _, file := range f.children {
		if file.name == name {
			return file
		}
	}
	return nil
}

func (parent *file) getChild(sub string) *file {
	child := parent.findChild(sub)
	if child == nil {
		child = &file{
			name:   sub,
			parent: parent,
			folder: &folder{
				sortAscending: []bool{true, true, true},
			},
		}
		parent.children = append(parent.children, child)
		parent.sorted = false
	}
	return child
}

func (app *app) analyze() {
	byHash := map[string][]*file{}
	app.analyzeRec(byHash, app.rootFolder)
	dups := map[string]struct{}{}
	for hash, files := range byHash {
		if len(files) > 1 {
			dups[hash] = struct{}{}
		}
	}
	for hash, files := range byHash {
		if _, ok := dups[hash]; ok {
			app.byHash[hash] = files
			for _, file := range files {
				file.dups = len(files)
			}
		}
	}
	app.nDuplicates = len(dups)
	app.rootFolder.updateMetas()
}

func (app *app) analyzeRec(byHash map[string][]*file, file *file) {
	if file.folder != nil {
		for _, child := range file.children {
			app.analyzeRec(byHash, child)
		}
	} else {
		files := byHash[file.hash]
		files = append(files, file)
		byHash[file.hash] = files
	}
}

func (app *app) findFile(path []string) *file {
	file := app.rootFolder
	for _, sub := range path {
		file = file.findChild(sub)
		if file == nil {
			return nil
		}
	}
	return file
}

func (app *app) getFile(path []string) *file {
	folder := app.rootFolder
	for _, sub := range path {
		folder = folder.getChild(sub)
	}
	return folder
}

func (f *file) path() (result []string) {
	for f.parent != nil {
		f = f.parent
		if f.name == "" {
			break
		}
		result = append(result, f.name)
	}
	slices.Reverse(result)
	return result
}

func (f *file) fullPath() (result []string) {
	result = append(result, f.name)
	for f.parent != nil {
		f = f.parent
		if f.name == "" {
			break
		}
		result = append(result, f.name)
	}
	slices.Reverse(result)
	return result
}

func (folder *file) updateMetas() {
	folder.size = 0
	folder.modTime = time.Time{}
	folder.dups = 0

	for _, child := range folder.children {
		if child.folder != nil {
			child.updateMetas()
		}
		folder.updateMeta(child)
	}
}

func (folder *file) updateMeta(meta *file) {
	folder.size += meta.size
	folder.dups += meta.dups
	if folder.modTime.Before(meta.modTime) {
		folder.modTime = meta.modTime
	}
}

func (app *app) deleteFile(file *file) {
	folder := app.findFile(file.path())
	folder.deleteFile(file)
}

func (folder *folder) deleteFile(file *file) {
	for childIdx, child := range folder.children {
		if child == file {
			folder.children = slices.Delete(folder.children, childIdx, childIdx+1)
			break
		}
	}
}
