package ui

import (
	"fmt"
	"html"
	"regexp"

	"github.com/conformal/gotk3/gtk"
	"github.com/tkerber/golem/cmd"
)

// keyMatcher matches a single already escaped "key" (i.e. value in angular
// brackets).
var keyMatcher = regexp.MustCompile(`&lt;.*?&gt;`)

// uriRegex matches groups of protocol, protocol seperator, domain and path.
var uriRegex = regexp.MustCompile(`^([^:]*)(:/{0,2})([^/]*)(/.*)$`)

// numRegex matches a decimal number.
var numRegex = regexp.MustCompile(`[0-9]+`)

// keysToMarkupString converts a slice of keys into an appropriate markup
// string.
func keysToMarkupString(keys []cmd.Key, selective, highlightNums bool) string {
	str := keyMatcher.ReplaceAllString(
		html.EscapeString(
			cmd.KeysStringSelective(keys, selective)),
		"<key>$0</key>")
	if highlightNums {
		return numRegex.ReplaceAllString(str, "<num>$0</num>")
	}
	return str
}

// A StatusBar contains the status bar UI elements.
type StatusBar struct {
	CmdStatus      *gtk.Label
	LocationStatus *gtk.Label
	container      gtk.Container
}

// SetLocationMarkup sets the text markup of the location.
func (s *StatusBar) SetLocationMarkup(label string) {
	GlibMainContextInvoke(s.LocationStatus.SetMarkup, label)
}

// SetCmdMarkup sets the text markup of the command status.
func (s *StatusBar) SetCmdMarkup(label string) {
	GlibMainContextInvoke(s.CmdStatus.SetMarkup, label)
}

// UpdateState updates the (command) state display of the window.
func (w *Window) UpdateState(state cmd.State) {
	var newStatus string
	switch s := state.(type) {
	case *cmd.NormalMode:
		// The status is either empty, or [current_binding] if it exists.
		if len(s.CurrentKeys) == 0 {
			newStatus = ""
		} else {
			newStatus = fmt.Sprintf(
				"[<em>%v</em>]",
				keysToMarkupString(s.CurrentKeys, true, true))
		}
	case *cmd.InsertMode:
		newStatus = "-- <em>insert</em> --"
	case *cmd.CommandLineMode:
		newStatus = fmt.Sprintf(
			":<em>%v</em>",
			keysToMarkupString(s.CurrentKeys, false, false))
	}
	w.SetCmdMarkup(w.MarkupReplacer.Replace(newStatus))
}

// UpdateLocation updates the location display of the window.
func (w *Window) UpdateLocation() {
	uri := w.GetURI()
	submatches := uriRegex.FindStringSubmatch(uri)
	var uriStr string
	if submatches == nil {
		uriStr = "<em>" + html.EscapeString(uri) + "</em>"
	} else {
		// https is highlighted green.
		if submatches[1] == "https" {
			uriStr += fmt.Sprintf("<secure>%s</secure>",
				html.EscapeString(submatches[1]))
		} else {
			uriStr += html.EscapeString(submatches[1])
		}
		// only the domain is otherwise emphasized.
		uriStr += fmt.Sprintf(
			"%s<em>%s</em>%s",
			html.EscapeString(submatches[2]),
			html.EscapeString(submatches[3]),
			html.EscapeString(submatches[4]),
		)
	}

	backForward := ""
	if w.CanGoBack() {
		backForward += "-"
	}
	if w.CanGoForward() {
		backForward += "+"
	}
	if backForward != "" {
		backForward = "[<em>" + backForward + "</em>]"
	}

	var pos string
	visible := int64(w.WebView.GetAllocatedHeight())
	if int64(visible) >= w.Height {
		pos = "all"
	} else if w.Top == 0 {
		pos = "top"
	} else if w.Top == w.Height-visible {
		pos = "bot"
	} else {
		percent := w.Top * 100 / (w.Height - visible)
		pos = fmt.Sprintf("%02d%%", percent)
	}

	locStr := fmt.Sprintf(
		"%s %s[<em>%d</em>/<em>%d</em>][<em>%s</em>]",
		uriStr,
		backForward,
		w.TabNumber,
		w.TabCount,
		pos,
	)
	w.SetLocationMarkup(w.MarkupReplacer.Replace(locStr))
}
