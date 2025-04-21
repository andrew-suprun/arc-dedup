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
		nDuplicates int
		hashing     int
		hashed      int
		state       appState

		folderTargets []folderTarget
		sortTargets   []sortTarget
		screenWidth   int
		screenHeight  int
		lastClickTime time.Time

		makeSelectedVisible bool
		sync                bool

		events events
	}

	file struct {
		name    string
		size    int
		modTime time.Time
		hash    string
		parent  *file
		*folder
	}

	files []*file

	folder struct {
		children      files
		selected      *file
		nFiles        int
		nHashed       int
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

func (app *app) getSelected() *file {
	return app.curFolder.getSelected()
}

func (app *app) setSelected(file *file) {
	app.curFolder = app.findFile(file.path())
	app.curFolder.selected = file
}

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
	if f.folder != nil {
		fmt.Fprintf(buf, ", nFiles: %d, nHashed: %d", f.nFiles, f.nHashed)
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

func (folder *file) getSelected() *file {
	if folder.selectedIdx >= len(folder.children) {
		folder.selectedIdx = len(folder.children) - 1
	}
	if folder.selectedIdx < 0 {
		folder.selectedIdx = 0
	}
	if folder.selected != nil {
		for i, child := range folder.children {
			if child == folder.selected {
				folder.selectedIdx = i
			}
		}
	}
	if len(folder.children) == 0 {
		return nil
	}
	folder.selected = folder.children[folder.selectedIdx]
	return folder.selected
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
	folder.nFiles = 0
	folder.nHashed = 0

	for _, child := range folder.children {
		if child.folder != nil {
			child.updateMetas()
			folder.nFiles += child.nFiles - 1
			folder.nHashed += child.nHashed
		}
		folder.updateMeta(child)
	}
	if folder.size == 0 && folder.parent != nil {
		folder.parent.deleteFile(folder)
	}
}

func (folder *file) updateMeta(meta *file) {
	folder.size += meta.size
	folder.nFiles++
	if meta.hash != "" {
		folder.nHashed++
	}
	if folder.modTime.Before(meta.modTime) {
		folder.modTime = meta.modTime
	}
}

type handleResult int

const (
	advance handleResult = iota
	stop
)

func (folder *file) walk(handle func(int, *file) handleResult) (result handleResult) {
	for idx, child := range folder.children {
		if child.folder != nil {
			result = child.walk(handle)
		} else {
			result = handle(idx, child)
		}
		if result == stop {
			break
		}
	}
	return result
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
