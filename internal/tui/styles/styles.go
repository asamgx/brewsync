package styles

import "github.com/charmbracelet/lipgloss"

// Catppuccin Mocha palette
var (
	CatRosewater = lipgloss.Color("#f5e0dc")
	CatFlamingo  = lipgloss.Color("#f2cdcd")
	CatPink      = lipgloss.Color("#f5c2e7")
	CatMauve     = lipgloss.Color("#cba6f7")
	CatRed       = lipgloss.Color("#f38ba8")
	CatMaroon    = lipgloss.Color("#eba0ac")
	CatPeach     = lipgloss.Color("#fab387")
	CatYellow    = lipgloss.Color("#f9e2af")
	CatGreen     = lipgloss.Color("#a6e3a1")
	CatTeal      = lipgloss.Color("#94e2d5")
	CatSky       = lipgloss.Color("#89dceb")
	CatSapphire  = lipgloss.Color("#74c7ec")
	CatBlue      = lipgloss.Color("#89b4fa")
	CatLavender  = lipgloss.Color("#b4befe")
	CatText      = lipgloss.Color("#cdd6f4")
	CatSubtext1  = lipgloss.Color("#bac2de")
	CatSubtext0  = lipgloss.Color("#a6adc8")
	CatOverlay2  = lipgloss.Color("#9399b2")
	CatOverlay1  = lipgloss.Color("#7f849c")
	CatOverlay0  = lipgloss.Color("#6c7086")
	CatSurface2  = lipgloss.Color("#585b70")
	CatSurface1  = lipgloss.Color("#45475a")
	CatSurface0  = lipgloss.Color("#313244")
	CatBase      = lipgloss.Color("#1e1e2e")
	CatMantle    = lipgloss.Color("#181825")
	CatCrust     = lipgloss.Color("#11111b")
)

// Colors - mapped to Catppuccin Mocha
var (
	PrimaryColor   = CatMauve    // Purple - active/selected
	SuccessColor   = CatGreen    // Green
	WarningColor   = CatPeach    // Peach/Orange
	ErrorColor     = CatRed      // Red
	MutedColor     = CatOverlay0 // Gray
	HighlightColor = CatMauve    // Purple
	BorderColor    = CatSurface1 // Border color
	TextColor      = CatText     // Main text
)

// Base styles
var (
	// Title style for headers
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryColor).
			MarginBottom(1)

	// Subtitle style
	SubtitleStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			MarginBottom(1)

	// Box style for panels
	BoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(MutedColor).
			Padding(1, 2)

	// Selected item style
	SelectedStyle = lipgloss.NewStyle().
			Foreground(SuccessColor).
			Bold(true)

	// Cursor style for current item
	CursorStyle = lipgloss.NewStyle().
			Foreground(HighlightColor).
			Bold(true)

	// Dimmed style for inactive items
	DimmedStyle = lipgloss.NewStyle().
			Foreground(MutedColor)

	// Added package style (+)
	AddedStyle = lipgloss.NewStyle().
			Foreground(SuccessColor)

	// Removed package style (-)
	RemovedStyle = lipgloss.NewStyle().
			Foreground(ErrorColor)

	// Warning style
	WarningStyle = lipgloss.NewStyle().
			Foreground(WarningColor)

	// Error style
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true)

	// Ignored package style
	IgnoredStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			Strikethrough(true)

	// Help style for keybindings
	HelpStyle = lipgloss.NewStyle().
			Foreground(MutedColor)

	// Category tab styles
	ActiveTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryColor).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(PrimaryColor).
			Padding(0, 1)

	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(MutedColor).
				Padding(0, 1)

	// Progress bar styles
	ProgressBarStyle = lipgloss.NewStyle().
				Foreground(PrimaryColor)

	ProgressCompleteStyle = lipgloss.NewStyle().
				Foreground(SuccessColor)

	// Status indicators
	CheckmarkStyle = lipgloss.NewStyle().
			Foreground(SuccessColor).
			SetString("✓")

	CrossStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			SetString("✗")

	WarningMarkStyle = lipgloss.NewStyle().
				Foreground(WarningColor).
				SetString("⚠")

	SpinnerStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor)

	// Border style for box drawing
	BorderStyle = lipgloss.NewStyle().
			Foreground(BorderColor)

	// Sidebar styles
	SidebarStyle = lipgloss.NewStyle().
			Foreground(TextColor)

	SidebarActiveStyle = lipgloss.NewStyle().
				Foreground(PrimaryColor).
				Bold(true)

	SidebarDimmedStyle = lipgloss.NewStyle().
				Foreground(MutedColor)

	// Header style
	HeaderStyle = lipgloss.NewStyle().
			Foreground(TextColor).
			Bold(true)

	// Footer style
	FooterStyle = lipgloss.NewStyle().
			Foreground(MutedColor)

	// Active indicator for sidebar
	ActiveIndicator = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			SetString("▌")

	// Separator line
	SeparatorStyle = lipgloss.NewStyle().
			Foreground(CatSurface2)
)

// Symbols
const (
	CheckedBox   = "[x]"
	UncheckedBox = "[ ]"
	Cursor       = ">"
	NoCursor     = " "
	Separator    = "─"
)

// CategoryColors maps package types to colors
var CategoryColors = map[string]lipgloss.Color{
	"tap":    lipgloss.Color("212"), // Blue
	"brew":   lipgloss.Color("42"),  // Green
	"cask":   lipgloss.Color("214"), // Yellow
	"vscode": lipgloss.Color("99"),  // Purple
	"cursor": lipgloss.Color("135"), // Light purple
	"go":     lipgloss.Color("39"),  // Cyan
	"mas":    lipgloss.Color("196"), // Red
}

// GetCategoryStyle returns a style for the given package type
func GetCategoryStyle(pkgType string) lipgloss.Style {
	if color, ok := CategoryColors[pkgType]; ok {
		return lipgloss.NewStyle().Foreground(color)
	}
	return lipgloss.NewStyle().Foreground(MutedColor)
}

// RenderCheckbox renders a checkbox with the given state
func RenderCheckbox(checked bool) string {
	if checked {
		return SelectedStyle.Render(CheckedBox)
	}
	return UncheckedBox
}

// RenderCursor renders a cursor or empty space
func RenderCursor(active bool) string {
	if active {
		return CursorStyle.Render(Cursor)
	}
	return NoCursor
}

// RenderDiff renders a diff indicator (+/-)
func RenderDiff(isAdd bool) string {
	if isAdd {
		return AddedStyle.Render("+")
	}
	return RemovedStyle.Render("-")
}
