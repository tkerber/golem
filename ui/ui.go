// Package ui contains structs and methods for interacting with the UI.
//
// Note that the term "ui" is used liberally here, and in general refers to
// any objects which have ui elements; most notably any webkit elements are
// considered "ui".
package ui

import (
	"fmt"

	"github.com/conformal/gotk3/gtk"
)

import "github.com/tkerber/golem/webkit"

// A UI Contains references to all significant UI objects.
type UI struct {
	StatusBar
	WebView *webkit.WebView
	Window  *gtk.Window
}

// NewUI Creates a new UI.
func NewUI() (*UI, error) {
	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		return nil, err
	}
	win.SetTitle("Golem")
	win.Connect("destroy", func() {
		gtk.MainQuit()
	})

	webView, err := webkit.NewWebView()
	if err != nil {
		return nil, err
	}

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

	ui := &UI{StatusBar{cmdStatus, locationStatus}, webView, win}

	webView.Connect("notify::title", func() {
		title := webView.GetTitle()
		if title != "" {
			win.SetTitle(fmt.Sprintf("%s - Golem", title))
		} else {
			win.SetTitle("Golem")
		}
	})
	webView.Connect("notify::uri", func() {
		ui.UpdateLocation()
	})
	bfl := webView.GetBackForwardList()
	bfl.Connect("changed", func() {
		ui.UpdateLocation()
	})

	return ui, nil
}

func (ui *UI) UpdateLocation() {
	locStr := ui.WebView.GetURI()
	backForward := ""
	if ui.WebView.CanGoBack() {
		backForward += "-"
	}
	if ui.WebView.CanGoForward() {
		backForward += "+"
	}
	if backForward != "" {
		locStr += " [" + backForward + "]"
	}
	ui.StatusBar.LocationStatus.SetLabel(locStr)
}

// StatusBar contains references to all significant status bar objects.
type StatusBar struct {
	CmdStatus      *gtk.Label
	LocationStatus *gtk.Label
}

func (s *StatusBar) SetCmdStatus(label string) {
	s.CmdStatus.SetLabel(label)
}
