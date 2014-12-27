package ui

// A ColorScheme encompasses the colors used in a Window.
type ColorScheme struct {
	FgEmphasized   Color
	FgUnemphasized Color
}

// A Color represents a single RGB color.
type Color uint32
