package ui

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/conformal/gotk3/gtk"
	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/golem/states"
	ggtk "github.com/tkerber/golem/gtk"
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
	CmdStatusLeft  *gtk.Label
	CmdStatusMid   *gtk.Label
	CmdStatusRight *gtk.Label
	LocationStatus *gtk.Label
	Container      gtk.Container
}

// SetLocationMarkup sets the text markup of the location.
func (s *StatusBar) SetLocationMarkup(label string) {
	ggtk.GlibMainContextInvoke(s.LocationStatus.SetMarkup, label)
}

// SetCmdMarkup sets the text markup of the command status.
func (s *StatusBar) SetCmdMarkup(left, mid, right string) {
	ggtk.GlibMainContextInvoke(s.CmdStatusLeft.SetMarkup, left)
	ggtk.GlibMainContextInvoke(s.CmdStatusMid.SetMarkup, mid)
	ggtk.GlibMainContextInvoke(s.CmdStatusRight.SetMarkup, right)
}

// UpdateState updates the (command) state display of the window.
func (w *Window) UpdateState(state cmd.State) {
	var newStatus string
	if comp, ok := state.(*cmd.CompletionMode); ok {
		state = (*comp.CompletionStates)[comp.CurrentCompletion]
	}
	switch s := state.(type) {
	case *cmd.NormalMode:
		var fmtStr string
		switch s.Substate {
		case states.NormalSubstateNormal:
			// The status is either empty, or [current_binding] if it exists.
			if len(s.CurrentKeys) == 0 {
				fmtStr = "%v"
			} else {
				fmtStr = "[<em>%v</em>]"
			}
		case states.NormalSubstateQuickmark:
			fmtStr = "Open quickmark: <em>%v</em>"
		case states.NormalSubstateQuickmarkTab:
			fmtStr = "Open quickmark in new tab: <em>%v</em>"
		case states.NormalSubstateQuickmarkWindow:
			fmtStr = "Open quickmark in new window: <em>%v</em>"
		case states.NormalSubstateQuickmarksRapid:
			fmtStr = "Open quickmarks in background: <em>%v</em>"
		}
		newStatus = fmt.Sprintf(fmtStr,
			keysToMarkupString(s.CurrentKeys, true, true))
	case *cmd.InsertMode:
		newStatus = "-- <em>insert</em> --"
	case *cmd.CommandLineMode:
		var substateStr string
		switch s.Substate {
		case states.CommandLineSubstateCommand:
			substateStr = ":"
		case states.CommandLineSubstateSearch:
			substateStr = "/"
		case states.CommandLineSubstateBackSearch:
			substateStr = "?"
		}
		beforeCursor := s.CurrentKeys[:s.CursorPos]
		afterCursor := s.CurrentKeys[s.CursorPos:]
		newStatus = fmt.Sprintf(
			"%s<em>%v</em><cursor>_</cursor><em>%v</em>",
			substateStr,
			keysToMarkupString(beforeCursor, false, false),
			keysToMarkupString(afterCursor, false, false))
	case *cmd.StatusMode:
		var fmtString string
		switch s.Substate {
		case states.StatusSubstateMinor:
			fmtString = "%s"
		case states.StatusSubstateMajor:
			fmtString = "<em>%s</em>"
		case states.StatusSubstateError:
			fmtString = "<error>%s</error>"
		}
		newStatus = fmt.Sprintf(fmtString, html.EscapeString(s.Status))
	case *cmd.ConfirmMode:
		newStatus = fmt.Sprintf(
			"%s <cursor>_</cursor>",
			html.EscapeString(s.Prompt))
	case *states.HintsMode:
		var action string
		switch s.Substate {
		case states.HintsSubstateFollow:
			action = "follow"
		case states.HintsSubstateBackground:
			action = "follow in background tab"
		case states.HintsSubstateRapid:
			action = "follow rapid"
		case states.HintsSubstateTab:
			action = "follow in new tab"
		case states.HintsSubstateWindow:
			action = "follow in new window"
		case states.HintsSubstateSearchEngine:
			action = "select search engine to add"
		}
		newStatus = fmt.Sprintf("%s: <em>%s</em>",
			action,
			keysToMarkupString(s.CurrentKeys, true, true))
	}
	split := strings.SplitN(newStatus, "<cursor>_</cursor>", 2)
	if len(split) == 1 {
		w.SetCmdMarkup(w.MarkupReplacer.Replace(split[0]), "", "")
	} else {
		w.SetCmdMarkup(
			w.MarkupReplacer.Replace(split[0]),
			w.MarkupReplacer.Replace("<cursor>_</cursor>"),
			w.MarkupReplacer.Replace(split[1]))
	}
}

// UpdateLocation updates the location display of the window.
func (w *Window) UpdateLocation() {
	wv := w.GetWebView()
	uri := wv.GetURI()
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
	if wv.CanGoBack() {
		backForward += "-"
	}
	if wv.CanGoForward() {
		backForward += "+"
	}
	if backForward != "" {
		backForward = "[<em>" + backForward + "</em>]"
	}

	load := wv.GetEstimatedLoadProgress()
	var loadStr string
	if load != 1.0 {
		loadStr = fmt.Sprintf("[<load>%02d%%</load>]", int(load*100))
	}

	markStr := ""
	if w.IsBookmarked() {
		markStr += "b"
	}
	if w.IsQuickmarked() {
		markStr += "q"
	}
	if markStr != "" {
		markStr = "[<em>" + markStr + "</em>]"
	}

	var pos string
	visible := int64(wv.GetAllocatedHeight())
	if int64(visible) >= w.GetHeight() {
		pos = "all"
	} else if w.GetTop() == 0 {
		pos = "top"
	} else if w.GetTop() == w.GetHeight()-visible {
		pos = "bot"
	} else {
		percent := w.GetTop() * 100 / (w.GetHeight() - visible)
		pos = fmt.Sprintf("%02d%%", percent)
	}

	locStr := fmt.Sprintf(
		"%s %s%s%s[<num>%d</num>/<num>%d</num>][<em>%s</em>]",
		uriStr,
		backForward,
		loadStr,
		markStr,
		w.TabNumber,
		w.TabCount,
		pos,
	)
	w.SetLocationMarkup(w.MarkupReplacer.Replace(locStr))
}
