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
		make(chan string, 8),
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

	statusBar, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		panic(fmt.Sprintf("Unable to create UI box: %v", err))
	}

	cmdStatus, err := gtk.LabelNew("")
	if err != nil {
		panic(fmt.Sprintf("Unable to create status label: %v", err))
	}
	//cmdStatus.SetUseMarkup(true)
	cmdStatus.OverrideFont("monospace")

	statusBar.PackStart(cmdStatus, false, false, 0)

	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		panic(fmt.Sprintf("Unable to create UI box: %v", err))
	}

	box.PackStart(webView, true, true, 0)
	box.PackStart(statusBar, false, false, 0)
	win.Add(box)

	win.SetDefaultSize(800, 600)
	win.ShowAll()

	_, err = glib.IdleAdd(func() bool {
		select {
		case i := <-cmdHandler.InstructionChan:
			err := i(webView)
			if err != nil {
				log.Printf("Command failed to execute: %v", err)
			}
		case s := <-cmdHandler.StatusChan:
			cmdStatus.SetLabel(s)
		// Very important so that this function doesn't block (and cause a
		// deadlock)
		default:
		}
		return true
	})
	if err != nil {
		panic("Failed to attach to glib event loop.")
	}

	gtk.Main()
}

func usage() {
	fmt.Printf("Usage: golem URI\n")
}
