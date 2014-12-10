package main

// #cgo pkg-config: glib-2.0 gobject-2.0
// #include <glib.h>
import "C"
import (
	"fmt"
	"os"

	"github.com/conformal/gotk3/gdk"
	"github.com/conformal/gotk3/gtk"

	"github.com/guelfey/go.dbus"
	"github.com/tkerber/golem/cfg"
	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/ipc"
	"github.com/tkerber/golem/ui"
)

func watchSignals(ch <-chan *dbus.Signal, ui *ui.UI) {
	const WE = "com.github.tkerber.golem.WebExtension"
	for signal := range ch {
		switch signal.Name {
		case WE + ".VerticalPositionChanged":
			ui.Top = signal.Body[0].(int64)
			ui.Height = signal.Body[1].(int64)
			ui.UpdateLocation()
		}
	}
}

func main() {

	if len(os.Args) == 2 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		usage()
		return
	}

	gtk.Init(nil)

	ui, err := ui.NewUI()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize UI: %v", err))
	}

	sessionBus, err := dbus.SessionBus()
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to DBus session bus: %v", err))
	}
	dbusObject := &ipc.WebExtension{sessionBus.Object(
		"com.github.tkerber.golem.WebExtension",
		"/com/github/tkerber/golem/WebExtension")}

	// Watch for signals on the proper interface 'n stuff.
	sessionBus.BusObject().Call(
		"org.freedesktop.DBus.AddMatch",
		0,
		"type='signal',"+
			"path='/com/github/tkerber/golem/WebExtension',"+
			"interface='com.github.tkerber.golem.WebExtension',"+
			"sender='com.github.tkerber.golem.WebExtension'")

	sigChan := make(chan *dbus.Signal, 100)
	sessionBus.Signal(sigChan)
	go watchSignals(sigChan, ui)

	settings := cfg.DefaultSettings

	cmdHandler := cmd.NewHandler(ui, settings, dbusObject)

	go cmdHandler.Run()

	if len(os.Args) > 1 {
		cmdHandler.RunCmd("open " + os.Args[1])
	} else {
		cmdHandler.RunCmd("open " + settings.HomePage)
	}

	ui.Window.Connect("key-press-event", func(w *gtk.Window, e *gdk.Event) bool {
		// This conversion *shouldn't* be unsafe, BUT we really don't want
		// crashes here. TODO
		e2 := gdk.EventKey{e}
		cmdHandler.KeyPressHandle <- e2.KeyVal()
		return <-cmdHandler.KeyPressSwallowChan
	})

	ui.Window.ShowAll()

	gtk.Main()
}

func usage() {
	fmt.Printf("Usage: golem [URI]\n")
}
