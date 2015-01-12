package golem

import (
	"fmt"
	"log"
	"time"

	"github.com/guelfey/go.dbus"
	"github.com/guelfey/go.dbus/introspect"
	"github.com/mattn/go-shellwords"
	"github.com/tkerber/golem/webkit"
)

const (
	// webExtenDBusInterface is the interface name of golem's web extensions.
	webExtenDBusInterface = "com.github.tkerber.golem.WebExtension"
	// webExtenDBusNamePrefix is a format string for the common prefix of the
	// names of all of golem's web extensions for a given profile.
	webExtenDBusNamePrefix = "com.github.tkerber.golem.WebExtension.%s.Page"
	// webExtenDBusName is a format string for the dbus name of the web
	// extension with the given profile and page id.
	webExtenDBusName = webExtenDBusNamePrefix + "%d"
	// webExtenDBusPathPrefix is a format string for the common prefix of the
	// paths used by golem's web extensions for a given profile.
	webExtenDBusPathPrefix = "/com/github/tkerber/golem/WebExtension/%s/page"
	// webExtenDBusPath is a format string for the dbus path of the web
	// extension with the given profile and page id.
	webExtenDBusPath = webExtenDBusPathPrefix + "%d"

	// webExtenWatchMessage is a format string for the message to watch dbus
	// signals from a particular web extension. It takes two profile, page id
	// pairs.
	webExtenWatchMessage = "type='signal',path='" + webExtenDBusPath +
		"',interface='" + webExtenDBusInterface +
		"',sender='" + webExtenDBusName + "'"

	// DBusInterface is the interface name of golem's main process.
	DBusInterface = "com.github.tkerber.Golem"
	// DBusName is a format string for the dbus name of golem's main
	// process, given the profile name as an argument.
	DBusName = "com.github.tkerber.Golem.%s"
	// DBusPath is the dbus path of golem's main process.
	DBusPath = "/com/github/tkerber/Golem"
	// DBusIntrospection is the introspection string of the interface of
	// golem's main process.
	DBusIntrospection = `
<node>
	<interface name="` + DBusInterface + `">
		<method name="NewWindow" />
		<method name="NewTabs">
			<arg direction="in" type="as" name="uris" />
		</method>
	</interface>` + introspect.IntrospectDataString + `</node>`
)

// DBusGolem is golem's DBus object.
type DBusGolem struct {
	golem *Golem
}

// CreateDBusWrapper creates the DBusGolem object for a concrete Golem
// instance.
func (g *Golem) CreateDBusWrapper() *DBusGolem {
	return &DBusGolem{g}
}

// NewWindow creates a new window in golem's main process.
func (g *DBusGolem) NewWindow() *dbus.Error {
	_, err := g.golem.NewWindow("")
	if err != nil {
		return &dbus.Error{
			fmt.Sprintf(DBusName+".Error", g.golem.profile),
			[]interface{}{err}}
	}
	return nil
}

// NewTabs opens a set of uris in new tabs.
func (g *DBusGolem) NewTabs(uris []string) *dbus.Error {
	// we try to split it into parts to allow searches to be passed
	// via command line. If this fails, we ignore the error and just
	// pass the whole string instead.
	for i, uri := range uris {
		parts, err := shellwords.Parse(uri)
		if err != nil {
			parts = []string{uri}
		}
		uris[i] = g.golem.OpenURI(parts)
	}
	w := g.golem.windows[0]
	_, err := w.NewTabs(uris...)
	if err != nil {
		return &dbus.Error{
			fmt.Sprintf(DBusName+".Error", g.golem.profile),
			[]interface{}{err}}
	}
	w.TabNext()
	return nil
}

// webExtension is the DBus object for a specific web extension.
type webExtension struct {
	*dbus.Object
}

// webExtensionForWebView creates a webExtension for a particular WebView.
func webExtensionForWebView(g *Golem, wv *webkit.WebView) *webExtension {
	page := wv.GetPageID()
	return &webExtension{g.sBus.Object(
		fmt.Sprintf(webExtenDBusName, g.profile, page),
		dbus.ObjectPath(fmt.Sprintf(webExtenDBusPath, g.profile, page)))}
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
	t := time.Now()
	call := w.Call(
		"org.freedesktop.DBus.Properties.Set",
		0,
		webExtenDBusInterface,
		"ScrollTop",
		dbus.MakeVariant(to))
	log.Println(time.Since(t))
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
