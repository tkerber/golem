package golem

import (
	"fmt"
	"math"

	"github.com/guelfey/go.dbus"
	"github.com/guelfey/go.dbus/introspect"
	"github.com/mattn/go-shellwords"
	"github.com/tkerber/golem/webkit"
)

const HintsChars = "FDSARTGBVECWXQZIOPMNHYULKJ"

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

// Blocks checks whether a uri is blocked by the adblocker or not.
func (g *DBusGolem) Blocks(uri, firstParty string, flags uint64) (bool, *dbus.Error) {
	return g.golem.adblocker.Blocks(uri, firstParty, flags), nil
}

// DomainElemHideCSS retrieves the css string to hide the elements on a given
// domain.
func (g *DBusGolem) DomainElemHideCSS(domain string) (string, *dbus.Error) {
	return g.golem.adblocker.DomainElemHideCSS(domain), nil
}

// GetHintsLabels gets n labels for hints.
func (g *DBusGolem) GetHintsLabels(n int64) ([]string, *dbus.Error) {
	ret := make([]string, n)
	if n == 0 {
		return ret, nil
	}
	length := int(math.Ceil(
		math.Log(float64(n)) / math.Log(float64(len(HintsChars)))))
	for i := range ret {
		bytes := make([]byte, length)
		divI := i
		for j := range bytes {
			bytes[j] = HintsChars[divI%len(HintsChars)]
			divI /= len(HintsChars)
		}
		ret[i] = string(bytes)
	}
	return ret, nil
}

// HintCall is called if a hint was hit.
func (g *DBusGolem) HintCall(uri string) (bool, *dbus.Error) {
	// TODO this is very much temporary.
	g.golem.windows[0].getWebView().LoadURI(uri)
	return false, nil
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

// getInt64 retrieves an int64 value.
func (w *webExtension) getInt64(name string) (int64, error) {
	v, err := w.GetProperty(webExtenDBusInterface + "." + name)
	if err != nil {
		return 0, err
	}
	return v.Value().(int64), nil
}

// getScrollTop retrieves the webExtension's scroll position from the top of
// the page.
func (w *webExtension) getScrollTop() (int64, error) {
	return w.getInt64("ScrollTop")
}

// getScrollLeft retrieves the webExtension's scroll position from the left of
// the page.
func (w *webExtension) getScrollLeft() (int64, error) {
	return w.getInt64("ScrollLeft")
}

// getScrollWidth retrieves the webExtension's scroll area width.
func (w *webExtension) getScrollWidth() (int64, error) {
	return w.getInt64("ScrollWidth")
}

// getScrollHeight retrieves the webExtension's scroll area height.
func (w *webExtension) getScrollHeight() (int64, error) {
	return w.getInt64("ScrollHeight")
}

// getScrollTargetTop retrieves the webExtension's scroll position from the
// top of the target scroll area.
func (w *webExtension) getScrollTargetTop() (int64, error) {
	return w.getInt64("ScrollTargetTop")
}

// getScrollTargetLeft retrieves the webExtension's scroll position from the
// left of the target scroll area.
func (w *webExtension) getScrollTargetLeft() (int64, error) {
	return w.getInt64("ScrollTargetLeft")
}

// getScrollTargetWidth retrieves the webExtension's target scroll area width.
func (w *webExtension) getScrollTargetWidth() (int64, error) {
	return w.getInt64("ScrollTargetWidth")
}

// getScrollTargetHeight retrieves the webExtension's target scroll area height.
func (w *webExtension) getScrollTargetHeight() (int64, error) {
	return w.getInt64("ScrollTargetHeight")
}

// setInf64 sets an int64 value.
func (w *webExtension) setInt64(name string, to int64) error {
	call := w.Call(
		"org.freedesktop.DBus.Properties.Set",
		0,
		webExtenDBusInterface,
		name,
		dbus.MakeVariant(to))
	return call.Err
}

// setScrollTop sets the webExtension's scroll position from the top.
func (w *webExtension) setScrollTop(to int64) error {
	return w.setInt64("ScrollTop", to)
}

// setScrollLeft sets the webExtension's scroll position from the left.
func (w *webExtension) setScrollLeft(to int64) error {
	return w.setInt64("ScrollLeft", to)
}

// setScrollTargetTop sets the webExtension's scroll position from the top of
// the target scroll area..
func (w *webExtension) setScrollTargetTop(to int64) error {
	return w.setInt64("ScrollTargetTop", to)
}

// setScrollTargetLeft sets the webExtension's scroll position from the left
// of the target scroll area.
func (w *webExtension) setScrollTargetLeft(to int64) error {
	return w.setInt64("ScrollTargetLeft", to)
}
