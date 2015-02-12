package golem

import (
	"fmt"
	"math"

	"github.com/guelfey/go.dbus"
	"github.com/guelfey/go.dbus/introspect"
	"github.com/mattn/go-shellwords"
	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/golem/states"
	"github.com/tkerber/golem/webkit"
)

type Session dbus.Conn

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
func (g *DBusGolem) HintCall(id uint64, uri string) (bool, *dbus.Error) {
	wv, ok := g.golem.webViews[id]
	if !ok {
		return false, &dbus.Error{
			fmt.Sprintf(DBusName+".Error", g.golem.profile),
			[]interface{}{
				"Invalid web page id recieved.",
			}}
	}
	w := wv.window
	if w == nil {
		return false, &dbus.Error{
			fmt.Sprintf(DBusName+".Error", g.golem.profile),
			[]interface{}{
				"WebView is not attached to any window.",
			}}
	}
	hm, ok := w.State.(*states.HintsMode)
	if !ok {
		return false, &dbus.Error{
			fmt.Sprintf(DBusName+".Error", g.golem.profile),
			[]interface{}{
				"Window not currently in hints mode.",
			}}
	}
	ret := hm.ExecuterFunction(uri)
	if ret == true {
		w.setState(&states.HintsMode{
			hm.StateIndependant,
			hm.Substate,
			hm.HintsCallback,
			nil,
			hm.ExecuterFunction,
		})
	}
	return ret, nil
}

// VerticalPositionChanged is called to signal a change in the vertical
// position of a web page.
func (g *DBusGolem) VerticalPositionChanged(
	id uint64, top, height int64) *dbus.Error {

	wv, ok := g.golem.webViews[id]
	if !ok {
		return &dbus.Error{
			fmt.Sprintf(DBusName+".Error", g.golem.profile),
			[]interface{}{
				"Invalid web page id recieved.",
			}}
	}
	wv.top = top
	wv.height = height
	for _, w := range g.golem.windows {
		if wv == w.getWebView() {
			w.UpdateLocation()
		}
	}
	return nil
}

// InputFocusChanged is called to signal a change in the input focus of a web
// page.
func (g *DBusGolem) InputFocusChanged(
	id uint64, focused bool) *dbus.Error {

	wv, ok := g.golem.webViews[id]
	if !ok {
		return &dbus.Error{
			fmt.Sprintf(DBusName+".Error", g.golem.profile),
			[]interface{}{
				"Invalid web page id recieved.",
			}}
	}
	// If it's newly focused, set any windows with this webview
	// displayed to insert mode.
	//
	// Otherwise, if the window is currently in insert mode and it's
	// newly unfocused, set this webview to normal mode.
	for _, w := range g.golem.windows {
		if wv == w.getWebView() {
			if focused {
				w.setState(
					cmd.NewInsertMode(w.State, cmd.SubstateDefault))
			} else if _, ok := w.State.(*cmd.InsertMode); ok {
				w.setState(
					cmd.NewNormalMode(w.State))
			}
		}
	}
	return nil
}

// webExtension is the DBus object for a specific web extension.
type webExtension struct {
	*dbus.Object
}

// webExtensionForWebView creates a webExtension for a particular WebView.
func webExtensionForWebView(g *Golem, wv *webkit.WebView) *webExtension {
	page := wv.GetPageID()
	return &webExtension{(*dbus.Conn)(g.Session).Object(
		fmt.Sprintf(webExtenDBusName, g.profile, page),
		dbus.ObjectPath(fmt.Sprintf(webExtenDBusPath, g.profile, page)))}
}

// LinkHintsMode initializes hints mode for links.
func (w *webExtension) LinkHintsMode() (int64, error) {
	call := w.Call(
		webExtenDBusInterface+".LinkHintsMode",
		dbus.FlagNoAutoStart)
	if call.Err != nil {
		return 0, call.Err
	}
	return call.Body[0].(int64), nil
}

// FormVariableHintsMode initializes hints mode for form input fields.
func (w *webExtension) FormVariableHintsMode() (int64, error) {
	call := w.Call(
		webExtenDBusInterface+".FormVariableHintsMode",
		dbus.FlagNoAutoStart)
	if call.Err != nil {
		return 0, call.Err
	}
	return call.Body[0].(int64), nil
}

// ClickHintsMode initializes hints mode for clickable elements.
func (w *webExtension) ClickHintsMode() (int64, error) {
	call := w.Call(
		webExtenDBusInterface+".ClickHintsMode",
		dbus.FlagNoAutoStart)
	if call.Err != nil {
		return 0, call.Err
	}
	return call.Body[0].(int64), nil
}

// EndHintsMode ends hints mode.
func (w *webExtension) EndHintsMode() error {
	call := w.Call(
		webExtenDBusInterface+".EndHintsMode",
		dbus.FlagNoAutoStart)
	return call.Err
}

// FilterHintsMode filters the displayed hints in hints mode.
//
// If a hint is matched precicely by a filter, it is hit.
func (w *webExtension) FilterHintsMode(filter string) (bool, error) {
	call := w.Call(
		webExtenDBusInterface+".FilterHintsMode",
		dbus.FlagNoAutoStart,
		filter)
	if call.Err != nil {
		return false, call.Err
	}
	return call.Body[0].(bool), nil
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
		dbus.FlagNoAutoStart,
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
