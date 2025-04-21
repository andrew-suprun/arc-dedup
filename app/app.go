package app

import (
	"dedup/fs"
	"log"
	"strings"

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

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		app.screenHeight = msg.Height
		app.screenWidth = msg.Width

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, tea.Quit
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
			folder.sorted = false
		}

	case fs.FileHashed:
		file := app.findFile(parsePath(msg.Path))
		file.hash = msg.Hash
		app.hashed++
		app.state = archiveHashing

	case fs.ArchiveHashed:
		log.Printf("got ArchiveHashed\n")
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
