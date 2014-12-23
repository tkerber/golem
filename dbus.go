package main

import (
	"fmt"

	"github.com/conformal/gotk3/gtk"
	"github.com/guelfey/go.dbus"
	"github.com/guelfey/go.dbus/introspect"
	"github.com/tkerber/golem/webkit"
)

const (
	webExtenDBusInterface  = "com.github.tkerber.golem.WebExtension"
	webExtenDBusNamePrefix = "com.github.tkerber.golem.WebExtension.Page"
	webExtenDBusName       = webExtenDBusNamePrefix + "%d"
	webExtenDBusPathPrefix = "/com/github/tkerber/golem/WebExtension/page"
	webExtenDBusPath       = webExtenDBusPathPrefix + "%d"

	webExtenWatchMessage = "type='signal',path='" + webExtenDBusPath +
		"',interface='" + webExtenDBusInterface +
		"',sender='" + webExtenDBusName + "'"

	golemDBusInterface     = "com.github.tkerber.Golem"
	golemDBusName          = "com.github.tkerber.Golem"
	golemDBusPath          = "/com/github/tkerber/Golem"
	golemDBusIntrospection = `
<node>
	<interface name="` + golemDBusInterface + `">
		<method name="NewWindow" />
	</interface>` + introspect.IntrospectDataString + `</node>`
)

type dbusGolem struct {
	*golem
}

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

type webExtension struct {
	*dbus.Object
}

func webExtensionForWebView(sBus *dbus.Conn, wv *webkit.WebView) *webExtension {
	page := wv.GetPageID()
	return &webExtension{sBus.Object(
		fmt.Sprintf(webExtenDBusName, page),
		dbus.ObjectPath(fmt.Sprintf(webExtenDBusPath, page)))}
}

func (w *webExtension) getScrollTop() (int64, error) {
	v, err := w.GetProperty(webExtenDBusInterface + ".ScrollTop")
	if err != nil {
		return 0, err
	}
	return v.Value().(int64), nil
}

func (w *webExtension) getScrollLeft() (int64, error) {
	v, err := w.GetProperty(webExtenDBusInterface + ".ScrollLeft")
	if err != nil {
		return 0, err
	}
	return v.Value().(int64), nil
}

func (w *webExtension) getScrollWidth() (int64, error) {
	v, err := w.GetProperty(webExtenDBusInterface + ".ScrollWidth")
	if err != nil {
		return 0, err
	}
	return v.Value().(int64), nil
}

func (w *webExtension) getScrollHeight() (int64, error) {
	v, err := w.GetProperty(webExtenDBusInterface + ".ScrollHeight")
	if err != nil {
		return 0, err
	}
	return v.Value().(int64), nil
}

func (w *webExtension) setScrollTop(to int64) error {
	call := w.Call(
		"org.freedesktop.DBus.Properties.Set",
		0,
		webExtenDBusInterface,
		"ScrollTop",
		dbus.MakeVariant(to))
	return call.Err
}

func (w *webExtension) setScrollLeft(to int64) error {
	call := w.Call(
		"org.freedesktop.DBus.Properties.Set",
		0,
		webExtenDBusInterface,
		"ScrollLeft",
		dbus.MakeVariant(to))
	return call.Err
}
