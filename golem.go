package main

import "github.com/conformal/gotk3/gtk"
import "github.com/tkerber/golem/webkit"
import "fmt"
import "os"

// For now, just a basic test. This browser opens one URI passed by
// command line argument.
func main() {
	if len(os.Args) != 2 {
		usage()
		return
	}
	gtk.Init(nil)
	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		panic(fmt.Sprintf("Unable to create window: %v", err))
	}
	win.SetTitle("Golem")
	win.Connect("destroy", func() {
		gtk.MainQuit()
	})

	webView, err := webkit.NewWebView()
	if err != nil {
		panic(fmt.Sprintf("Unable to create webview: %v", err))
	}
	webView.LoadURI(os.Args[1])
	win.Add(webView)

	win.SetDefaultSize(800, 600)
	win.ShowAll()

	gtk.Main()
}

func usage() {
	fmt.Printf("Usage: golem URI\n")
}
