package main

import (
	"fmt"

	"github.com/conformal/gotk3/gtk"
	"github.com/guelfey/go.dbus"
	"github.com/guelfey/go.dbus/introspect"
	"github.com/tkerber/golem/webkit"
)

const (
	// webExtenDBusInterface is the interface name of golem's web extensions.
	webExtenDBusInterface = "com.github.tkerber.golem.WebExtension"
	// webExtenDBusNamePrefix is the common prefix of the names of all of
	// golem's web extensions.
	webExtenDBusNamePrefix = "com.github.tkerber.golem.WebExtension.Page"
	// webExtenDBusName is a format string for the dbus name of the web
	// extension with the given page id.
	webExtenDBusName = webExtenDBusNamePrefix + "%d"
	// webExtenDBusPathPrefix is the common prefix of the paths used by golem's
	// web extensions.
	webExtenDBusPathPrefix = "/com/github/tkerber/golem/WebExtension/page"
	// webExtenDBusPath is a format string for the dbus path of the web
	// extension with the given page id.
	webExtenDBusPath = webExtenDBusPathPrefix + "%d"

	// webExtenWatchMessage is a format string for the message to watch dbus
	// signals from a particular web extension. It takes the page id twice.
	webExtenWatchMessage = "type='signal',path='" + webExtenDBusPath +
		"',interface='" + webExtenDBusInterface +
		"',sender='" + webExtenDBusName + "'"

	// golemDBusInterface is the interface name of golem's main process.
	golemDBusInterface = "com.github.tkerber.Golem"
	// golemDBusName is the dbus name of golem's main process.
	golemDBusName = "com.github.tkerber.Golem"
	// golemDBusPath is the dbus path of golem's main process.
	golemDBusPath = "/com/github/tkerber/Golem"
	// golemDBusIntrospection is the introspection string of the interface of
	// golem's main process.
	golemDBusIntrospection = `
<node>
	<interface name="` + golemDBusInterface + `">
		<method name="NewWindow" />
	</interface>` + introspect.IntrospectDataString + `</node>`
)

// dbusGolem is golem's DBus object.
type dbusGolem struct {
	*golem
}

// NewWindow creates a new window in golem's main process.
func (g *dbusGolem) NewWindow() *dbus.Error {
	// GTK can't be running during creation of new window, lest it
	// crash and burn horrificly. (Remeber, this is triggered by DBus, not
	// GTK. Consider this a kind of join() with the GTK thread)
	gtk.MainQuit()
	// Aww, [defer go gtk.Main()] doesn't work :(
	defer func() { go gtk.Main() }()

	err := g.newWindow(g.defaultSettings)
	if err != nil {
		return &dbus.Error{golemDBusName + ".Error", []interface{}{err}}
	}
	return nil
}

// webExtension is the DBus object for a specific web extension.
type webExtension struct {
	*dbus.Object
}

// webExtensionForWebView creates a webExtension for a particular WebView.
func webExtensionForWebView(sBus *dbus.Conn, wv *webkit.WebView) *webExtension {
	page := wv.GetPageID()
	return &webExtension{sBus.Object(
		fmt.Sprintf(webExtenDBusName, page),
		dbus.ObjectPath(fmt.Sprintf(webExtenDBusPath, page)))}
}

// getScrollTop retrieves the webExtension's scroll position from the top of
// the page.
func (w *webExtension) getScrollTop() (int64, error) {
	v, err := w.GetProperty(webExtenDBusInterface + ".ScrollTop")
	if err != nil {
		return 0, err
	}
	return v.Value().(int64), nil
}

// getScrollLeft retrieves the webExtension's scroll position from the left of
// the page.
func (w *webExtension) getScrollLeft() (int64, error) {
	v, err := w.GetProperty(webExtenDBusInterface + ".ScrollLeft")
	if err != nil {
		return 0, err
	}
	return v.Value().(int64), nil
}

// getScrollWidth retrieves the webExtension's scroll area width.
func (w *webExtension) getScrollWidth() (int64, error) {
	v, err := w.GetProperty(webExtenDBusInterface + ".ScrollWidth")
	if err != nil {
		return 0, err
	}
	return v.Value().(int64), nil
}

// getScrollHeight retrieves the webExtension's scroll area height.
func (w *webExtension) getScrollHeight() (int64, error) {
	v, err := w.GetProperty(webExtenDBusInterface + ".ScrollHeight")
	if err != nil {
		return 0, err
	}
	return v.Value().(int64), nil
}

// setScrollTop sets the webExtension's scroll position from the top.
func (w *webExtension) setScrollTop(to int64) error {
	call := w.Call(
		"org.freedesktop.DBus.Properties.Set",
		0,
		webExtenDBusInterface,
		"ScrollTop",
		dbus.MakeVariant(to))
	return call.Err
}

// setScrollLeft sets the webExtension's scroll position from the left.
func (w *webExtension) setScrollLeft(to int64) error {
	call := w.Call(
		"org.freedesktop.DBus.Properties.Set",
		0,
		webExtenDBusInterface,
		"ScrollLeft",
		dbus.MakeVariant(to))
	return call.Err
}
