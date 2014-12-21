package main

import (
	"fmt"

	"github.com/conformal/gotk3/gtk"
	"github.com/guelfey/go.dbus"
	"github.com/guelfey/go.dbus/introspect"
)

func main() {
	// Try to acquire the golem bus
	sBus, err := dbus.SessionBus()
	if err != nil {
		panic(fmt.Sprintf("Failed to acquire session bus: %v", err))
	}
	repl, err := sBus.RequestName(
		"com.github.tkerber.Golem",
		dbus.NameFlagDoNotQueue)
	if err != nil {
		panic(fmt.Sprintf("Failed to ascertain status of Golem's bus name."))
	}
	switch repl {
	// If we get it, this is the new golem. Hurrah!
	case dbus.RequestNameReplyPrimaryOwner:
		// TODO do we want GTK argument parsing?
		gtk.Init(nil)
		webkitInit()
		g, err := newGolem()
		if err != nil {
			panic(fmt.Sprintf("Error during golem initialization: %v", err))
		}
		sBus.Export(
			&dbusGolem{g},
			"/com/github/tkerber/Golem",
			"com.github.tkerber.Golem")
		sBus.Export(
			introspect.Introspectable(golemDBusIntrospection),
			"/com/github/tkerber/Golem",
			"org.freedesktop.DBus.Introspectable")
		g.newWindow()
		// This doesn't need to run in a goroutine, but as the gtk main
		// loop can be stopped and restarted in a goroutine, this makes
		// more sense.
		go gtk.Main()
		<-g.quit
		sBus.ReleaseName("com.github.tkerber.Golem")
	// If not, we attach to the existing one.
	default:
		o := sBus.Object(
			"com.github.tkerber.Golem",
			"/com/github/tkerber/Golem")
		o.Call(
			"com.github.tkerber.Golem.NewWindow",
			0)
	}
}
