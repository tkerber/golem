package ui

import (
	"fmt"
	"strings"
)

// A ColorScheme encompasses the colors used in a Window.
//
// Its CSS parameter is the CSS string to be used with the color scheme.
//
// Its MarkupReplacer is a Replacer which "converts" an internal markup
// representation into pango's text markup.
//
// Internal tags are <em> (for emphasis), <secure> and <key>.
type ColorScheme struct {
	FgEmphasized   Color
	FgUnemphasized Color
	FgSecure       Color
	FgKey          Color
	Bg             Color
	MarkupReplacer *strings.Replacer
	CSS            string
}

const cssFormatString = `
GtkBox {
	background-color: #%06x;
	color: #%06x;
}`

// NewColorScheme creates a new color scheme, given the specified colors.
func NewColorScheme(
	emphasized,
	unemphasized,
	secure,
	key,
	bg Color) *ColorScheme {

	return &ColorScheme{
		emphasized,
		unemphasized,
		secure,
		key,
		bg,
		strings.NewReplacer(
			"<em>",
			fmt.Sprintf(`<span color="#%06x">`, emphasized),
			"</em>",
			"</span>",
			"<secure>",
			fmt.Sprintf(`<span color="#%06x">`, secure),
			"</secure>",
			"</span>",
			"<key>",
			fmt.Sprintf(`<span color="#%06x">`, key),
			"</key>",
			"</span>",
		),
		fmt.Sprintf(cssFormatString, bg, unemphasized),
	}
}

// A Color represents a single RGB color.
type Color uint32
