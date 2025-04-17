package app

import (
	"dedup/fs"
	"dedup/lifecycle"
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func Run(fsys fs.FS) {
	m := make(model, 1)
	p := tea.NewProgram(m)

	app := &app{
		fs:     fsys,
		lc:     lifecycle.New(),
		events: events{p},
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
			file := app.getFile(parsePath(meta.Path))
			file.hash = meta.Hash
			file.modTime = meta.ModTime
			file.size = meta.Size
			if meta.Hash == "" {
				app.hashing++
			}
		}

	case fs.FileHashed:
		file := app.findFile(parsePath(msg.Path))
		file.hash = msg.Hash
		app.hashed++
		app.state = archiveScanned

	case fs.ArchiveHashed:
		app.state = archiveHashed
		app.analyzeArchives()
	}
	return m, nil
}

func (m model) View() string {
	return ""
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
