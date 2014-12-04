// Package ui contains structs and methods for interacting with the UI.
//
// Note that the term "ui" is used liberally here, and in general refers to
// any objects which have ui elements; most notably any webkit elements are
// considered "ui".
package ui

import "github.com/conformal/gotk3/gtk"

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
	webView.Connect("notify::title", func() {
		win.SetTitle(webView.GetTitle() + " - Golem")
	})

	statusBar, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return nil, err
	}

	cmdStatus, err := gtk.LabelNew("")
	if err != nil {
		return nil, err
	}
	cmdStatus.OverrideFont("monospace")

	statusBar.PackStart(cmdStatus, false, false, 0)

	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, err
	}

	box.PackStart(webView, true, true, 0)
	box.PackStart(statusBar, false, false, 0)
	win.Add(box)

	win.SetDefaultSize(800, 600)

	return &UI{StatusBar{cmdStatus}, webView, win}, nil
}

// StatusBar contains references to all significant status bar objects.
type StatusBar struct {
	CmdStatus *gtk.Label
}

func (s *StatusBar) SetCmdStatus(label string) {
	s.CmdStatus.SetLabel(label)
}
