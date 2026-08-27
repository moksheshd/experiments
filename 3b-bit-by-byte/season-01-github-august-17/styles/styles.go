// Package styles is the shared terminal theme for the 3B: Bit By Byte,
// Season One labs. Every episode imports it so the whole season reads as one
// series in the terminal. Keep it small and restrained: a few roles, not a
// rainbow.
package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Palette. ANSI-256 codes chosen to stay legible on both light and dark
// terminals. lipgloss downgrades or drops color automatically when the output
// is not a terminal (for example when piped to a file), so data output stays
// clean.
var (
	Accent = lipgloss.Color("39")  // blue: structure, indices
	Good   = lipgloss.Color("42")  // green: revealed answers, healthy
	Warn   = lipgloss.Color("214") // amber: the prompt, attention
	Bad    = lipgloss.Color("203") // red: failure, exhaustion
	Muted  = lipgloss.Color("245") // gray: secondary text
)

// Roles. Prefer these over ad-hoc styles so episodes stay consistent.
var (
	Title = lipgloss.NewStyle().Bold(true).Foreground(Accent)
	Index = lipgloss.NewStyle().Bold(true).Foreground(Accent)
	Bold  = lipgloss.NewStyle().Bold(true)
	Sub   = lipgloss.NewStyle().Foreground(Muted)
	OK    = lipgloss.NewStyle().Foreground(Good)
	Attn  = lipgloss.NewStyle().Foreground(Warn)
	Err   = lipgloss.NewStyle().Foreground(Bad)

	panel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Accent).
		Padding(0, 2)
)

// Header renders a rounded, accent-bordered banner: a bold title with an
// optional muted meta line beneath it.
func Header(title, meta string) string {
	inner := Title.Render(title)
	if meta != "" {
		inner += "\n" + Sub.Render(meta)
	}
	return panel.Render(inner)
}

// Rule returns a muted horizontal rule n columns wide.
func Rule(n int) string {
	return Sub.Render(strings.Repeat("─", n))
}
