// Package ui contains golem's user-interface implementation.
package ui

// #cgo pkg-config: gtk+-3.0
// #include <gtk/gtk.h>
// #include <stdlib.h>
import "C"
import (
	"errors"
	"runtime"
	"unsafe"

	"github.com/conformal/gotk3/glib"
	"github.com/conformal/gotk3/gtk"
	"github.com/conformal/gotk3/pango"
	ggtk "github.com/tkerber/golem/gtk"
)

var errNilPtr = errors.New("Unexpected nil pointer.")

// A Window is one of golem's windows.
type Window struct {
	*StatusBar
	*TabBar
	*CompletionBar
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

	statusBar, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return nil, err
	}
	statusBar.SetName("statusbar")

	cmdStatii := make([]*gtk.Label, 3)
	for i := range cmdStatii {
		cmdStatii[i], err = gtk.LabelNew("")
		if err != nil {
			return nil, err
		}
		cmdStatii[i].OverrideFont("monospace")
		cmdStatii[i].SetUseMarkup(true)
	}
	cmdStatii[0].SetEllipsize(pango.ELLIPSIZE_START)
	cmdStatii[2].SetEllipsize(pango.ELLIPSIZE_END)
	cmdStatii[2].SetMarginEnd(5)

	locationStatus, err := gtk.LabelNew("")
	if err != nil {
		return nil, err
	}
	locationStatus.OverrideFont("monospace")
	locationStatus.SetUseMarkup(true)
	locationStatus.SetEllipsize(pango.ELLIPSIZE_START)
	locationStatus.SetMarginStart(5)

	for _, status := range cmdStatii {
		statusBar.PackStart(status, false, false, 0)
	}
	statusBar.PackEnd(locationStatus, false, false, 0)

	statusBarEventBox, err := gtk.EventBoxNew()
	if err != nil {
		return nil, err
	}
	statusBarEventBox.Add(statusBar)
	w.StatusBar = &StatusBar{
		cmdStatii[0],
		cmdStatii[1],
		cmdStatii[2],
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

	completions, err := w.newCompletionBar()
	if err != nil {
		return nil, err
	}
	w.CompletionBar = completions

	mainOverlayPtr := unsafe.Pointer(C.gtk_overlay_new())
	if mainOverlayPtr == nil {
		return nil, errNilPtr
	}
	mainOverlay := &gtk.Container{gtk.Widget{glib.InitiallyUnowned{
		&glib.Object{glib.ToGObject(mainOverlayPtr)}}}}
	mainOverlay.Object.RefSink()
	runtime.SetFinalizer(mainOverlay.Object, (*glib.Object).Unref)
	mainOverlay.Add(webViewStack)
	C.gtk_overlay_add_overlay(
		(*C.GtkOverlay)(mainOverlayPtr),
		(*C.GtkWidget)(unsafe.Pointer(completions.Container.Native())))

	contentBox, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return nil, err
	}
	contentBox.PackStart(tabBar.ScrolledWindow, false, false, 0)
	contentBox.PackStart(mainOverlay, true, true, 0)

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

// ToggleStatusBar toggles the visibility of the status bar.
func (w *Window) ToggleStatusBar() {
	ggtk.GlibMainContextInvoke(func() {
		if w.StatusBar.Container.IsVisible() {
			w.StatusBar.Container.Hide()
		} else {
			w.StatusBar.Container.Show()
		}
	})
}

// ToggleTabBar toggles the visibility of the tab bar.
func (w *Window) ToggleTabBar() {
	ggtk.GlibMainContextInvoke(func() {
		if w.TabBar.IsVisible() {
			w.TabBar.Hide()
		} else {
			w.TabBar.Show()
		}
	})
}

// ToggleUI toggles all UI (non-webkit) elements.
func (w *Window) ToggleUI() {
	w.ToggleStatusBar()
	w.ToggleTabBar()
}

// HideUI hides all UI (non-webkit) elements.
func (w *Window) HideUI() {
	ggtk.GlibMainContextInvoke(func() {
		w.StatusBar.Container.Hide()
		w.TabBar.Hide()
	})
}

// ShowUI shows all UI elements.
func (w *Window) ShowUI() {
	ggtk.GlibMainContextInvoke(func() {
		w.StatusBar.Container.Show()
		w.TabBar.Show()
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
