package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	styleDefault         = lipgloss.NewStyle().Foreground(lipgloss.Color("17")).Background(lipgloss.Color("250"))
	styleScreenTooSmall  = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Background(lipgloss.Color("9")).Bold(true)
	styleArchive         = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Background(lipgloss.Color("0")).Bold(true).Italic(true)
	styleBreadcrumbs     = lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("17")).Bold(true)
	styleFile            = lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("17"))
	styleFileDup         = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Background(lipgloss.Color("17"))
	styleFileSelected    = lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("19"))
	styleFileDupSelected = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Background(lipgloss.Color("19"))
	styleFolderHeader    = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("243")).Bold(true)
	styleProgressBar     = lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("33")).Bold(true)
)

type builder struct {
	app          *app
	builder      strings.Builder
	x, y         int
	style        lipgloss.Style
	markX, markY int
}

func (app *app) render() string {
	app.targets = app.targets[:0]
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
	b.app.targets = b.app.targets[:0]

	path := b.app.curFolder.fullPath()

	b.markPosition()
	b.text(" Root")
	b.setTarget(selectFolder{})

	for i, name := range path {
		b.text(" / ")
		b.markPosition()
		b.text(name)
		selectPath := make([]string, i+1)
		copy(selectPath, path[:i+1])
		b.setTarget(selectFolder{path: selectPath})
	}

	b.newLine()
}

func (b *builder) renderFolder() {
	document := "     Document" + b.app.curFolder.sortIndicator(sortByName)
	modified := "Date Modified" + b.app.curFolder.sortIndicator(sortByTime)
	size := "Size" + b.app.curFolder.sortIndicator(sortBySize)
	b.setStyle(styleFolderHeader)
	b.text(padRight(document, b.app.screenWidth-39))
	b.text(padRight(modified, 20))
	b.text(padLeft(size, 18))
	b.newLine()

	folder := b.app.curFolder
	for i := range b.app.screenHeight - 4 {
		if i+folder.offsetIdx >= len(folder.children) {
			b.setStyle(styleFile)
			b.newLine()
		} else {
			b.markPosition()
			file := folder.children[i+folder.offsetIdx]
			if b.app.curFolder.selectedIdx == i+b.app.curFolder.offsetIdx {
				if file.dups > 0 {
					b.setStyle(styleFileDupSelected)
				} else {
					b.setStyle(styleFileSelected)
				}
			} else {
				b.setStyle(styleFile)
				if file.dups > 0 {
					b.setStyle(styleFileDup)
				} else {
					b.setStyle(styleFile)
				}
			}
			if file.dups > 0 {
				if file.folder != nil {
					b.text(" D ")
				} else {
					b.text(counter(file.dups))
				}
			} else {
				b.text("   ")
			}
			if file.folder == nil {
				b.text("  ")
			} else {
				b.text("▶ ")
			}
			b.text(padRight(file.name, b.app.screenWidth-45))
			b.text(file.modTime.Format(" 2006-01-02 15:04:05"))
			b.text(formatSize(file.size))
			b.text(" ")
			b.setTarget(selectFile{idx: i + folder.offsetIdx})
			b.newLine()
		}
	}
	b.setStyle(styleArchive)
	switch b.app.state {
	case archiveScanning:
		b.text(" Scanning ")
		b.text(padRight("", b.app.screenWidth-b.x))
	case archiveHashing:
		b.text(" Hashing ")
		b.setStyle(styleProgressBar)
		b.text(b.progressBar(b.app.hashed, b.app.hashing, b.app.screenWidth-10))
		b.setStyle(styleArchive)
		b.text(" ")
	case archiveReady:
		b.text(" All Clear ")
		b.text(padRight("", b.app.screenWidth-b.x))
	}
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
	for b.y < b.app.screenHeight {
		b.newLine()
	}
	return b.builder.String()
}

func (b *builder) markPosition() {
	b.markX, b.markY = b.x, b.y
}

func (b *builder) setTarget(cmd any) {
	b.app.targets = append(b.app.targets, target{x1: b.markX, x2: b.x, y1: b.markY, y2: b.y, cmd: cmd})
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
	b.y++
}

func (b *builder) progressBar(done, size, width int) string {
	runes := make([]rune, width)
	value := 0
	if size > 0 {
		value = (done*width*8 + size/2) / size
	}
	idx := 0
	for ; idx < value/8; idx++ {
		runes[idx] = '█'
	}
	if value%8 > 0 {
		runes[idx] = []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉'}[value%8]
		idx++
	}
	for ; idx < int(width); idx++ {
		runes[idx] = ' '
	}
	return b.style.Render(string(runes))
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

func padRight(text string, size int) string {
	runes := []rune(text)
	if len(runes) > size {
		return string(runes[:size])
	}
	for len(runes) < size {
		runes = append(runes, ' ')
	}
	return string(runes)
}

func padLeft(text string, size int) string {
	runes := []rune(text)
	if len(runes) > size {
		return string(runes[len(runes)-size:])
	}
	padded := []rune{}
	for len(padded)+len(runes) < size {
		padded = append(padded, ' ')
	}
	padded = append(padded, runes...)
	return string(padded)
}

func formatSize(size int) string {
	str := fmt.Sprintf("%15d", size)
	slice := []string{str[:3], str[3:6], str[6:9], str[9:12]}
	b := strings.Builder{}
	for _, s := range slice {
		b.WriteString(s)
		if s == " " || s == "   " {
			b.WriteString(" ")
		} else {
			b.WriteString(",")
		}
	}
	b.WriteString(str[12:])
	return b.String()
}

func counter(count int) string {
	if count > 9 {
		return " * "
	}
	return fmt.Sprintf(" %c ", '0'+count)
}
