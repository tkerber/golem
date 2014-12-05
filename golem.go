package main

// #cgo pkg-config: glib-2.0 gobject-2.0
// #include <glib.h>
import (
	"C"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/conformal/gotk3/gdk"
	"github.com/conformal/gotk3/gtk"

	"github.com/tkerber/golem/cfg"
	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/ui"
)

func main() {

	tmpDir, err := ioutil.TempDir(os.TempDir(), "golem")
	if err != nil {
		panic(fmt.Sprintf("Failed to allocated temporary directory: %v", err))
	}
	os.Setenv("GOLEM_TMP", tmpDir)
	defer os.RemoveAll(tmpDir)

	if len(os.Args) == 2 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		usage()
		return
	}

	gtk.Init(nil)

	ui, err := ui.NewUI()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize UI: %v", err))
	}

	settings := cfg.DefaultSettings

	cmdHandler := cmd.NewHandler(ui, settings)

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
