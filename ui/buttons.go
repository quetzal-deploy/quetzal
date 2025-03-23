package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type Button interface {
	Keybind() string
	Render(m model) string
	RenderWithBaseStyle(m model, baseStyle lipgloss.Style) string
}

type MenuItem struct {
	name    string
	keybind string
}

func (mi MenuItem) Keybind() string {
	return mi.keybind
}

func (mi MenuItem) Render(m model) string {
	return mi.RenderWithBaseStyle(m, lipgloss.NewStyle())
}

func (mi MenuItem) RenderWithBaseStyle(m model, baseStyle lipgloss.Style) string {
	r := strings.Builder{}

	style := baseStyle.Foreground(menuAccentColor)

	before, after, found := strings.Cut(mi.name, mi.keybind)

	if found {
		r.WriteString(baseStyle.Render(before))
		r.WriteString(style.Render(mi.keybind))
		r.WriteString(baseStyle.Render(after))

	} else {
		r.WriteString(baseStyle.Render(mi.name + " ("))
		r.WriteString(style.Render(mi.keybind))
		r.WriteString(baseStyle.Render(")"))
	}

	return r.String()
}

type PauseButton struct {
	keybind string
}

func (pb PauseButton) Keybind() string {
	return pb.keybind
}

func (pb PauseButton) Render(m model) string {
	return pb.RenderWithBaseStyle(m, lipgloss.NewStyle())
}

func (pb PauseButton) RenderWithBaseStyle(m model, baseStyle lipgloss.Style) string {
	text := " pause"
	keybindName := keybindDisplayName(pb.keybind)

	keybindStyle := baseStyle.Foreground(menuAccentColor)
	textHighlightStyle := baseStyle.Foreground(lipgloss.Color("#FFFFFF")).Background(menuAccentColor)

	if m.paused {
		text = "paused"
	}

	r := strings.Builder{}

	blinkOn := time.Now().Second()%2 == 0

	if m.paused && blinkOn {
		r.WriteString(textHighlightStyle.Render(text))
	} else {
		r.WriteString(baseStyle.Render(text))
	}

	// render keybind in ()
	r.WriteString(baseStyle.Render(" ("))
	r.WriteString(keybindStyle.Render(keybindName))
	r.WriteString(baseStyle.Render(")"))

	return r.String()
}
