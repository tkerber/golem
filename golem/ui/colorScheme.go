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
// Internal tags are <em> (for emphasis), <secure>, <key>, <num>, <focus>,
// <load>, <cursor> and <error>.
type ColorScheme struct {
	FgEmphasized   Color
	FgUnemphasized Color
	FgError        Color
	FgSecure       Color
	FgKey          Color
	FgCursor       Color
	Num            Color
	FgFocus        Color
	BgFocus        Color
	FgLoad         Color
	Bg             Color
	TabBarBg       Color
	MarkupReplacer *strings.Replacer
	CSS            string
}

const cssFormatString = `
GtkBox#statusbar, GtkBox#tabbar GtkLabel {
	background-color: #%06x;
	color: #%06x;
}
GtkBox#tabbar {
	background-color: #%06x;
}`

// NewColorScheme creates a new color scheme, given the specified colors.
func NewColorScheme(
	emphasized,
	unemphasized,
	err,
	secure,
	key,
	cursor,
	num,
	fgFocus,
	bgFocus,
	load,
	bg,
	tabbarBg Color) *ColorScheme {

	return &ColorScheme{
		emphasized,
		unemphasized,
		err,
		secure,
		key,
		cursor,
		num,
		fgFocus,
		bgFocus,
		load,
		bg,
		tabbarBg,
		strings.NewReplacer(
			"<em>",
			fmt.Sprintf(`<span color="#%06x">`, emphasized),
			"</em>",
			"</span>",
			"<error>",
			fmt.Sprintf(`<span color="#%06x">`, err),
			"</error>",
			"</span>",
			"<secure>",
			fmt.Sprintf(`<span color="#%06x">`, secure),
			"</secure>",
			"</span>",
			"<key>",
			fmt.Sprintf(`<span color="#%06x">`, key),
			"</key>",
			"</span>",
			"<num>",
			fmt.Sprintf(`<span color="#%06x">`, num),
			"</num>",
			"</span>",
			"<focus>",
			fmt.Sprintf(`<span bgcolor="#%06x" color="#%06x">`,
				bgFocus,
				fgFocus),
			"</focus>",
			"</span>",
			"<load>",
			fmt.Sprintf(`<span color="#%06x">`, load),
			"</load>",
			"</span>",
			"<cursor>",
			fmt.Sprintf(`<span color="#%06x">`, cursor),
			"</cursor>",
			"</span>",
		),
		fmt.Sprintf(cssFormatString, bg, unemphasized, tabbarBg),
	}
}

// A Color represents a single RGB color.
type Color uint32
