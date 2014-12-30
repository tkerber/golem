package main

// #cgo pkg-config: webkit2gtk-4.0
// #include <webkit2/webkit2.h>
// #include <stdlib.h>
import "C"
import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/conformal/gotk3/glib"
	"github.com/tkerber/golem/webkit"
)

// webkitInit initializes webkit for golem's use.
func (g *golem) webkitInit() {
	extenDir, err := ioutil.TempDir("", "golem-web-exten")
	if err != nil {
		panic("Failed to create temporary directory.")
	}
	g.extenDir = extenDir
	extenData, err := Asset("libgolem.so")
	if err != nil {
		panic("Failed to access web extension embedded data.")
	}
	extenPath := filepath.Join(extenDir, "libgolem.so")
	err = ioutil.WriteFile(extenPath, extenData, 0700)
	if err != nil {
		panic("Failed to write web extension to temporary directory.")
	}

	c := webkit.GetDefaultWebContext()
	c.SetWebExtensionsDirectory(extenDir)

	// Set the profile string to be passed to the web extensions.
	cProfile := C.CString(g.profile)
	defer C.free(unsafe.Pointer(cProfile))
	profileVariant := C.g_variant_new_string((*C.gchar)(cProfile))
	C.webkit_web_context_set_web_extensions_initialization_user_data(
		(*C.WebKitWebContext)(unsafe.Pointer(c.Native())),
		profileVariant)

	// NOTE: removing this will cause bugs in golems web extension.
	// Tread lightly.
	c.SetProcessModel(webkit.ProcessModelMultipleSecondaryProcesses)

	c.SetCacheModel(webkit.CacheModelWebBrowser)
	c.SetDiskCacheDirectory(g.files.cacheDir)

	c.GetCookieManager().SetPersistentStorage(
		g.files.cookies,
		webkit.CookiePersistentStorageText)

	c.Connect("download-started", func(_ *glib.Object, d *webkit.Download) {
		// Find the window
		wv := d.GetWebView()
		var win *window
	outer:
		for _, w := range g.windows {
			for _, wv2 := range w.webViews {
				if wv.Native() == wv2.Native() {
					win = w
					break outer
				}
			}
		}
		if win != nil {
			win.addDownload(d)
		}
		g.addDownload(d)
		dlDir := g.files.downloadDir
		d.Connect("decide-destination", func(d *webkit.Download, suggestedName string) bool {
			// Check if the file with the suggested name exists in dlDir
			path := filepath.Join(dlDir, suggestedName)
			_, err := os.Stat(path)
			exists := !os.IsNotExist(err)
			for i := 1; exists; i++ {
				path = filepath.Join(dlDir, fmt.Sprintf("%d_%s", i, suggestedName))
				_, err := os.Stat(path)
				exists = !os.IsNotExist(err)
			}
			d.SetDestination(fmt.Sprintf("file://%s", path))
			return false
		})
	})

	// TODO this is temporary.
	c.RegisterURIScheme("golem", golemSchemeHandler)
}

// webkitCleanup removes the temporary webkit extension directory.
func (g *golem) webkitCleanup() {
	os.RemoveAll(g.extenDir)
}

// golemSchemeHandler handles request to the 'golem:' scheme.
func golemSchemeHandler(req *webkit.URISchemeRequest) {
	req.Finish([]byte("<html><head><title>Golem</title></head><body><h1>Golem Home Page</h1><p>And stuff.</p></body></html>"), "text/html")
}
