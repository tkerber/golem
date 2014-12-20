package main

import (
	"github.com/conformal/gotk3/gtk"
	"github.com/guelfey/go.dbus"
	"github.com/guelfey/go.dbus/introspect"
)

const golemDBusInterface = "com.github.tkerber.Golem"
const golemDBusName = "com.github.tkerber.Golem"
const golemDBusPath = "/com/github/tkerber/Golem"
const golemDBusIntrospection = `
<node>
	<interface name="` + golemDBusInterface + `">
		<method name="NewWindow" />
	</interface>` + introspect.IntrospectDataString + `</node>`

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

	err := g.newWindow()
	if err != nil {
		return &dbus.Error{golemDBusName + ".Error", []interface{}{err}}
	}
	return nil
}
