package main

// #cgo pkg-config: glib-2.0 gobject-2.0
// #include <glib.h>
import "C"
import "github.com/conformal/gotk3/gtk"
import "github.com/conformal/gotk3/gdk"
import "github.com/conformal/gotk3/glib"
import "github.com/tkerber/golem/webkit"
import "github.com/tkerber/golem/cmd"
import "fmt"
import "os"
import "log"

var gsrc *C.GSource = nil

// command line argument.
func main() {
	if len(os.Args) != 2 {
		usage()
		return
	}

	cmdHandler := &cmd.Handler{
		make(chan uint),
		make(chan bool),
		make(chan cmd.Instruction, 8),
	}

	go cmdHandler.Run()

	gtk.Init(nil)

	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		panic(fmt.Sprintf("Unable to create window: %v", err))
	}
	win.SetTitle("Golem")
	win.Connect("destroy", func() {
		gtk.MainQuit()
	})
	win.Connect("key-press-event", func(w *gtk.Window, e *gdk.Event) bool {
		// This conversion *shouldn't* be unsafe, BUT we really don't want
		// crashes here. TODO
		e2 := gdk.EventKey{e}
		cmdHandler.KeyPressHandle <- e2.KeyVal()
		return <-cmdHandler.KeyPressSwallowChan
	})

	webView, err := webkit.NewWebView()
	if err != nil {
		panic(fmt.Sprintf("Unable to create webview: %v", err))
	}

	webView.LoadURI(os.Args[1])
	win.Add(webView)

	win.SetDefaultSize(800, 600)
	win.ShowAll()

	sh, err := glib.IdleAdd(func() bool {
		select {
		case i := <-cmdHandler.InstructionChan:
			err := i(webView)
			if i != nil {
				log.Printf("Command failed to execute: %v", err)
			}
		default:
		}
		return true
	})
	if err != nil {
		panic("Failed to attach to glib event loop.")
	}
	gsrc = C.g_main_context_find_source_by_id(nil, C.guint(sh))
	C.g_source_ref(gsrc)

	gtk.Main()
}

func usage() {
	fmt.Printf("Usage: golem URI\n")
}
