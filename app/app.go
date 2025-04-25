package app

import (
	"dedup/fs"
	"log"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func Run(fsys fs.FS) {
	m := make(model, 1)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	rootFolder := &file{
		folder: &folder{
			sortAscending: []bool{true, true, true},
		},
	}

	app := &app{
		fs:         fsys,
		rootFolder: rootFolder,
		curFolder:  rootFolder,
		byHash:     map[string][]*file{},
		events:     events{p},
	}

	fsys.Scan(app.events)

	m <- app

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type events struct {
	p *tea.Program
}

func (e events) Send(event any) {
	e.p.Send(event)
}

type model chan *app

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	app := <-m
	defer func() { m <- app }()
	defer func() {
		if app.curFolder.selectedIdx >= len(app.curFolder.children) {
			app.curFolder.selectedIdx = len(app.curFolder.children) - 1
		}
		if app.curFolder.selectedIdx < 0 {
			app.curFolder.selectedIdx = 0
		}
		if app.curFolder.offsetIdx >= len(app.curFolder.children)+4-app.screenHeight {
			app.curFolder.offsetIdx = len(app.curFolder.children) + 4 - app.screenHeight
		}
		if app.curFolder.offsetIdx < 0 {
			app.curFolder.offsetIdx = 0
		}
	}()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		app.screenHeight = msg.Height
		app.screenWidth = msg.Width

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, tea.Quit
		case "up":
			app.curFolder.selectedIdx--

		case "down":
			app.curFolder.selectedIdx++

		case "pgup":
			app.curFolder.selectedIdx -= app.screenHeight - 4
			app.curFolder.offsetIdx -= app.screenHeight - 4

		case "pgdown":
			app.curFolder.selectedIdx += app.screenHeight - 4
			app.curFolder.offsetIdx += app.screenHeight - 4

		case "home":
			app.curFolder.selectedIdx = 0
			app.curFolder.offsetIdx = 0

		case "end":
			app.curFolder.selectedIdx = len(app.curFolder.children) - 1
			app.curFolder.offsetIdx = len(app.curFolder.children) + 4 - app.screenHeight

		case "left":
			if app.curFolder.parent != nil {
				app.curFolder = app.curFolder.parent
			}
		case "right":
			child := app.curFolder.children[app.curFolder.selectedIdx]
			if child.folder != nil {
				app.curFolder = child
				break
			}

		case "tab":
			file := app.curFolder.children[app.curFolder.selectedIdx]
			if file.folder != nil {
				break
			}
			files := app.byHash[file.hash]
			if len(files) < 2 {
				break
			}
			i := 0
			for ; files[i] != file; i++ {
			}
			nextFile := files[(i+1)%len(files)]
			app.curFolder = nextFile.parent
			for idx, child := range app.curFolder.children {
				if child == nextFile {
					app.curFolder.selectedIdx = idx
					break
				}
			}
		case "enter":
			file := app.curFolder.children[app.curFolder.selectedIdx]
			if file.folder != nil {
				break
			}
			files := app.byHash[file.hash]
			if len(files) < 2 {
				break
			}
			for _, dup := range files {
				if dup != file {
					app.fs.Remove(filepath.Join(dup.fullPath()...))
					app.deleteFile(dup)
				}
			}
			for idx, child := range app.curFolder.children {
				if child == file {
					app.curFolder.selectedIdx = idx
					break
				}
			}
			app.analyze()
		}

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			for _, target := range app.targets {
				if msg.X >= target.x1 && msg.X <= target.x2 && msg.Y >= target.y1 && msg.Y <= target.y2 {
					switch cmd := target.cmd.(type) {
					case selectFolder:
						app.curFolder = app.findFile(cmd.path)

					case selectFile:
						app.curFolder.selectedIdx = cmd.idx
						file := app.curFolder.children[cmd.idx]
						if file.folder != nil &&
							app.lastX == msg.X && app.lastY == msg.Y &&
							time.Since(app.lastClickTime).Milliseconds() < 500 {

							app.curFolder = file
						}
						app.lastClickTime = time.Now()
						app.lastX = msg.X
						app.lastY = msg.Y

					case sortCmd:
						if cmd.column == app.curFolder.sortColumn {
							app.curFolder.sortAscending[cmd.column] = !app.curFolder.sortAscending[cmd.column]
						} else {
							app.curFolder.sortColumn = cmd.column
						}
						app.curFolder.sort()
					}
					break
				}
			}
		} else if msg.Button == tea.MouseButtonWheelUp {
			app.curFolder.offsetIdx--
		} else if msg.Button == tea.MouseButtonWheelDown {
			app.curFolder.offsetIdx++
		}

	case fs.FileMetas:
		for _, meta := range msg {
			path, name := parseName(meta.Path)
			incoming := &file{
				name:    name,
				size:    meta.Size,
				modTime: meta.ModTime,
				hash:    meta.Hash,
			}
			folder := app.getFile(path)
			folder.children = append(folder.children, incoming)
			incoming.parent = folder
			if meta.Hash == "" {
				app.hashing++
			}
		}
		app.rootFolder.updateMetas()
		app.rootFolder.sortRec()

	case fs.FileHashed:
		file := app.findFile(parsePath(msg.Path))
		file.hash = msg.Hash
		app.hashed++
		app.state = archiveHashing

	case fs.ArchiveHashed:
		app.state = archiveReady
		app.analyze()
	}
	return m, nil
}

func (m model) View() string {
	app := <-m
	result := app.render()
	m <- app
	return result

}

func parsePath(strPath string) []string {
	if strPath == "" {
		return nil
	}
	return strings.Split(strPath, "/")
}

func parseName(strPath string) ([]string, string) {
	path := parsePath(strPath)
	base := path[len(path)-1]
	return path[:len(path)-1], base
}
