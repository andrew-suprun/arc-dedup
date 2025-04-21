package app

import (
	"log"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	styleDefault        = lipgloss.NewStyle().Foreground(lipgloss.Color("17")).Background(lipgloss.Color("250"))
	styleScreenTooSmall = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Background(lipgloss.Color("9")).Bold(true)
	styleArchive        = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Background(lipgloss.Color("0")).Bold(true).Italic(true)
	styleBreadcrumbs    = lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("17")).Bold(true)
	styleFolderHeader   = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("243")).Bold(true)
	styleProgressBar    = lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("33")).Bold(true)
)

type builder struct {
	app          *app
	builder      strings.Builder
	x, y         int
	style        lipgloss.Style
	markX, markY int
}

func (app *app) render() string {
	b := builder{app: app, builder: strings.Builder{}}

	if app.screenWidth < 80 || app.screenHeight < 24 {
		return b.renderTooSmall()
	}

	b.renderTitle()
	b.renderBreadcrumbs()
	b.renderFolder()
	return b.builder.String()
}

func (b *builder) renderTitle() {
	b.setStyle(styleArchive)
	b.text(" ")
	b.text(b.app.fs.Root())
	b.newLine()
}

func (b *builder) renderBreadcrumbs() {
	b.setStyle(styleBreadcrumbs)
	b.app.folderTargets = b.app.folderTargets[:0]

	path := b.app.curFolder.fullPath()

	b.text(" Root")

	for _, name := range path {
		b.text(" / ")
		b.text(name)
	}

	b.newLine()
}

func (b *builder) renderFolder() {
	b.setStyle(styleDefault)
	b.setStyle(styleFolderHeader)
	b.text(" State ")
	b.text("Document", b.app.curFolder.sortIndicator(sortByName))
	log.Println("ind", len(b.app.curFolder.sortIndicator(sortByName)))
	// b.text("Document")
	b.skipTo(b.app.screenWidth - 39)
	b.text("Date Modified", b.app.curFolder.sortIndicator(sortByTime))
	sortBySizeIndicator := b.app.curFolder.sortIndicator(sortBySize)
	b.skipTo(b.app.screenWidth - 5 - len(sortBySizeIndicator))
	b.text("Size", sortBySizeIndicator, " ")
	b.newLine()
}

func (b *builder) renderTooSmall() string {
	b.setStyle(styleScreenTooSmall)
	for range b.app.screenHeight / 2 {
		b.newLine()
	}
	y := (b.app.screenWidth - 12) / 2
	for range y {
		b.text(" ")
	}
	b.text("Too Small...")
	b.newLine()
	for range b.app.screenHeight / 2 {
		b.newLine()
	}
	return b.builder.String()
}

func (b *builder) markPosition() {
	b.markX, b.markY = b.x, b.y
}

func (b *builder) setTarget(func(app *app)) {
}

func (b *builder) setStyle(style lipgloss.Style) {
	b.style = style
}

func (b *builder) text(texts ...string) {
	for _, text := range texts {
		runes := []rune(text)
		b.x += len(runes)
		b.builder.WriteString(b.style.Render(text))
	}
}

func (b *builder) newLine() {
	for b.x < b.app.screenWidth {
		b.builder.WriteString(b.style.Render(" "))
		b.x++
	}
	b.builder.WriteString(b.style.Render("\n"))
	b.x = 0
}

func (b *builder) skipTo(x int) {
	for b.x < x {
		b.builder.WriteString(b.style.Render(" "))
		b.x++
	}
}

func (f *folder) sortIndicator(column sortColumn) string {
	if column == f.sortColumn {
		if f.sortAscending[column] {
			return " ▲"
		}
		return " ▼"
	}
	return ""
}
