package ui

import "github.com/charmbracelet/lipgloss"

var (
	Brand   = lipgloss.Color("#6366f1")
	Success = lipgloss.Color("#22c55e")
	Warning = lipgloss.Color("#f59e0b")
	Error   = lipgloss.Color("#ef4444")
	Muted   = lipgloss.Color("#6b7280")
	White   = lipgloss.Color("#f9fafb")

	BrandStyle = lipgloss.NewStyle().Foreground(Brand).Bold(true)
	SuccessStyle = lipgloss.NewStyle().Foreground(Success)
	ErrorStyle   = lipgloss.NewStyle().Foreground(Error)
	WarningStyle = lipgloss.NewStyle().Foreground(Warning)
	MutedStyle   = lipgloss.NewStyle().Foreground(Muted)
	BoldStyle    = lipgloss.NewStyle().Bold(true)

	Banner = lipgloss.NewStyle().
		Foreground(Brand).
		Bold(true).
		Padding(0, 1)

	BoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Brand).
		Padding(0, 1)

	CheckMark = SuccessStyle.Render("✓")
	CrossMark = ErrorStyle.Render("✗")
	Arrow     = BrandStyle.Render("→")
)

func Check(ok bool) string {
	if ok {
		return CheckMark
	}
	return CrossMark
}
