// Package ui contains golem's user-interface implementation.
package ui

// #cgo pkg-config: gtk+-3.0
// #include <gtk/gtk.h>
// #include <stdlib.h>
import "C"
import (
	"errors"
	"unsafe"

	"github.com/conformal/gotk3/gtk"
	"github.com/conformal/gotk3/pango"
	ggtk "github.com/tkerber/golem/gtk"
)

// A Window is one of golem's windows.
type Window struct {
	*StatusBar
	*TabBar
	WebView
	*gtk.Window
	*ColorScheme
	Callback
	webViewStack *gtk.Stack
	// The number of the active tab.
	TabNumber int
	// The number of total tabs in this window.
	TabCount int
}

// NewWindow creates a new window containing the given WebView.
func NewWindow(webView WebView, callback Callback) (*Window, error) {
	rets := ggtk.GlibMainContextInvoke(newWindow, webView, callback)
	if rets[1] != nil {
		return nil, rets[1].(error)
	}
	return rets[0].(*Window), nil
}

// newWindow creates a new window containing the given WebView.
//
// MUST BE CALLED IN GLIB'S MAIN CONTEXT.
func newWindow(webView WebView, callback Callback) (*Window, error) {
	colors := NewColorScheme(
		0xffffff,
		0x888888,
		0xff8888,
		0xaaffaa,
		0xffaa88,
		0xff8888,
		0x66aaaa,
		0xdddddd,
		0x225588,
		0xdd9955,
		0x333333,
		0x222222,
	)

	w := &Window{
		nil,
		nil,
		webView,
		nil,
		colors,
		callback,
		nil,
		1,
		1,
	}

	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		return nil, err
	}
	win.SetTitle("Golem")
	w.Window = win

	sp := C.gtk_css_provider_new()
	css := colors.CSS
	gErr := new(*C.GError)
	cCSS := C.CString(css)
	defer C.free(unsafe.Pointer(cCSS))
	C.gtk_css_provider_load_from_data(
		sp,
		(*C.gchar)(cCSS),
		-1,
		gErr)
	if *gErr != nil {
		goStr := C.GoString((*C.char)((**gErr).message))
		C.g_error_free(*gErr)
		return nil, errors.New(goStr)
	}
	screen, err := win.GetScreen()
	if err != nil {
		return nil, err
	}
	C.gtk_style_context_add_provider_for_screen(
		(*C.GdkScreen)(unsafe.Pointer(screen.Native())),
		(*C.GtkStyleProvider)(unsafe.Pointer(sp)),
		C.GTK_STYLE_PROVIDER_PRIORITY_APPLICATION)

	statusBar, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	if err != nil {
		return nil, err
	}
	statusBar.SetName("statusbar")

	cmdStatus, err := gtk.LabelNew("")
	if err != nil {
		return nil, err
	}
	cmdStatus.OverrideFont("monospace")
	cmdStatus.SetUseMarkup(true)
	cmdStatus.SetEllipsize(pango.ELLIPSIZE_START)

	locationStatus, err := gtk.LabelNew("")
	if err != nil {
		return nil, err
	}
	locationStatus.OverrideFont("monospace")
	locationStatus.SetUseMarkup(true)
	locationStatus.SetEllipsize(pango.ELLIPSIZE_START)

	statusBar.PackStart(cmdStatus, false, false, 0)
	statusBar.PackEnd(locationStatus, false, false, 0)

	statusBarEventBox, err := gtk.EventBoxNew()
	if err != nil {
		return nil, err
	}
	statusBarEventBox.Add(statusBar)
	w.StatusBar = &StatusBar{
		cmdStatus,
		locationStatus,
		statusBarEventBox.Container}

	tabBar, err := NewTabBar(w)
	if err != nil {
		return nil, err
	}
	w.TabBar = tabBar

	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, err
	}

	webViewStack, err := gtk.StackNew()
	if err != nil {
		return nil, err
	}
	w.webViewStack = webViewStack
	webViewStack.Add(webView.GetWebView())

	contentBox, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return nil, err
	}
	contentBox.PackStart(tabBar, false, false, 0)
	contentBox.PackStart(webViewStack, true, true, 0)

	box.PackStart(contentBox, true, true, 0)
	box.PackStart(statusBarEventBox, false, false, 0)
	win.Add(box)

	// TODO sensible default size. (Default to screen size?)
	win.SetDefaultSize(800, 600)

	return w, nil
}

// Show shows the window.
func (w *Window) Show() {
	ggtk.GlibMainContextInvoke(w.Window.ShowAll)
}

// HideUI hides all UI (non-webkit) elements.
func (w *Window) HideUI() {
	ggtk.GlibMainContextInvoke(func() {
		w.StatusBar.Container.Hide()
		w.TabBar.Box.Hide()
	})
}

// ShowUI shows all UI elements.
func (w *Window) ShowUI() {
	ggtk.GlibMainContextInvoke(func() {
		w.StatusBar.Container.Show()
		w.TabBar.Box.Show()
	})
}

// SetTitle wraps gtk.Window.SetTitle in glib's main context.
func (w *Window) SetTitle(title string) {
	ggtk.GlibMainContextInvoke(w.Window.SetTitle, title)
}

// SwitchToWebView switches the shown web view.
func (w *Window) SwitchToWebView(wv WebView) {
	ggtk.GlibMainContextInvoke(func() {
		wvWidget := wv.GetWebView()
		w.webViewStack.SetVisibleChild(wvWidget)
		w.WebView = wv
		wvWidget.GrabFocus()
	})
}

// AttachWebView connects a web view to the window, but doesn't show it yet.
func (w *Window) AttachWebView(wv WebView) {
	ggtk.GlibMainContextInvoke(func() {
		wv := wv.GetWebView()
		w.webViewStack.Add(wv)
		wv.Show()
	})
}
