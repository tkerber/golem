package ui

import (
	"fmt"

	"github.com/conformal/gotk3/gtk"
	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/webkit"
)

type Window struct {
	StatusBar
	*webkit.WebView
	*gtk.Window
	top    int64
	height int64
}

func NewWindow(webView *webkit.WebView) (*Window, error) {
	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		return nil, err
	}
	win.SetTitle("Golem")

	statusBar, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return nil, err
	}

	cmdStatus, err := gtk.LabelNew("")
	if err != nil {
		return nil, err
	}
	cmdStatus.OverrideFont("monospace")

	locationStatus, err := gtk.LabelNew("")
	if err != nil {
		return nil, err
	}
	locationStatus.OverrideFont("monospace")

	statusBar.PackStart(cmdStatus, false, false, 0)
	statusBar.PackEnd(locationStatus, false, false, 0)

	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, err
	}

	box.PackStart(webView, true, true, 0)
	box.PackStart(statusBar, false, false, 0)
	win.Add(box)

	// TODO sensible default size. (Default to screen size?)
	win.SetDefaultSize(800, 600)

	w := &Window{StatusBar{cmdStatus, locationStatus}, webView, win, 0, 0}

	return w, nil
}

func (w *Window) Show() {
	w.Window.ShowAll()
}

func (w *Window) UpdateState(state cmd.State) {
	var newStatus string
	switch s := state.(type) {
	case *cmd.NormalMode:
		// The status is either empty, or [current_binding] if it exists.
		if len(s.CurrentKeys) == 0 {
			newStatus = ""
		} else {
			newStatus = fmt.Sprintf("[%v]", cmd.KeysString(s.CurrentKeys))
		}
	case *cmd.InsertMode:
		newStatus = "--INSERT--"
	case *cmd.CommandLineMode:
		newStatus = fmt.Sprintf(
			":%v",
			cmd.KeysStringSelective(s.CurrentKeys, false))
	}
	w.SetCmdLabel(newStatus)
}

func (w *Window) UpdateLocation() {
	locStr := w.GetURI()
	locStr += " "

	backForward := ""
	if w.CanGoBack() {
		backForward += "-"
	}
	if w.CanGoForward() {
		backForward += "+"
	}
	if backForward != "" {
		locStr += "[" + backForward + "]"
	}

	var pos string
	visible := int64(w.WebView.GetAllocatedHeight())
	if int64(visible) >= w.height {
		pos = "ALL"
	} else if w.top == 0 {
		pos = "TOP"
	} else if w.top == w.height-visible {
		pos = "BOT"
	} else {
		percent := w.top * 100 / (w.height - visible)
		pos = fmt.Sprintf("%02d%%", percent)
	}
	locStr += "[" + pos + "]"

	w.SetLocationLabel(locStr)
}
