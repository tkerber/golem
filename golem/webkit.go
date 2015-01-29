package golem

// #cgo pkg-config: webkit2gtk-4.0
// #include <webkit2/webkit2.h>
// #include <stdlib.h>
import "C"
import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unsafe"

	"github.com/conformal/gotk3/glib"
	"github.com/tkerber/golem/webkit"
)

var urlMatchRegex = regexp.MustCompile(`(http://|https://|file:///).*`)

// webkitInit initializes webkit for golem's use.
func (g *Golem) webkitInit() {
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
	c.SetFaviconDatabaseDirectory("")

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
		wv, err := d.GetWebView()
		// Download has no webview. It is probably a silent download, so we
		// drop it.
		if err != nil {
			return
		}
		var win *Window
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
		d.Connect("decide-destination",
			func(d *webkit.Download, suggestedName string) bool {
				// Check if the file with the suggested name exists in dlDir
				path := filepath.Join(dlDir, suggestedName)
				_, err := os.Stat(path)
				exists := !os.IsNotExist(err)
				for i := 1; exists; i++ {
					path = filepath.Join(
						dlDir,
						fmt.Sprintf("%d_%s", i, suggestedName))
					_, err := os.Stat(path)
					exists = !os.IsNotExist(err)
				}
				d.SetDestination(fmt.Sprintf("file://%s", path))
				return false
			})
	})

	c.RegisterURIScheme("golem-unsafe", golemUnsafeSchemeHandler)
	c.RegisterURIScheme("golem", golemSchemeHandler)
	c.GetSecurityManager().RegisterURISchemeAsLocal("golem")
}

// WebkitCleanup removes the temporary webkit extension directory.
func (g *Golem) WebkitCleanup() {
	os.RemoveAll(g.extenDir)
}

// golemSchemeHandler handles request to the 'golem:' scheme.
func golemSchemeHandler(req *webkit.URISchemeRequest) {
	req.FinishError(errors.New("Invalid request"))
}

// golemUnsafeSchemeHandler handles request to the 'golem-unsafe:' scheme.
func golemUnsafeSchemeHandler(req *webkit.URISchemeRequest) {
	rPath := strings.TrimPrefix(req.GetURI(), "golem-unsafe://")
	// If we have a ? or # suffix, we discard it.
	splitPath := strings.SplitN(rPath, "#", 2)
	rPath = splitPath[0]
	splitPath = strings.SplitN(rPath, "?", 2)
	rPath = splitPath[0]
	data, err := Asset(path.Join("srv", rPath))
	if err == nil {
		mime := guessMimeFromExtension(rPath)
		req.Finish(data, mime)
	} else {
		switch {
		case strings.HasPrefix(rPath, "pdf.js/loop/"):
			handleLoopRequest(req)
		default:
			// TODO finish w/ error
			req.FinishError(errors.New("Invalid request"))
		}
	}
}

// handleLoopRequest handles a request to a loop/ pseudo-file
//
// Any request to a golem protocol containing the path /loop/ (which is not an
// existing asset) will be treated as a loop request, with the URI after the
// loop/ part.
// E.g. golem-unsafe:///pdf.js/loop/http://example.com/example-pdf.pdf
func handleLoopRequest(req *webkit.URISchemeRequest) {
	// We loop a page request from another scheme into the golem scheme
	// Ever-so-slightly dangerous.
	splitPath := strings.SplitN(req.GetURI(), "/loop/", 2)
	if len(splitPath) == 1 {
		req.Finish([]byte{}, "text/plain")
		return
	}
	uri := splitPath[1]

	if !urlMatchRegex.MatchString(uri) {
		req.FinishError(fmt.Errorf("Invalid url: %v", uri))
		return
	}

	tmpDir, err := ioutil.TempDir("", "golem-loop-")
	if err != nil {
		req.FinishError(err)
		return
	}
	dlFile, err := filepath.Abs(filepath.Join(tmpDir, "loopdl"))
	if err != nil {
		req.FinishError(err)
		os.RemoveAll(tmpDir)
		return
	}
	dwnld := webkit.GetDefaultWebContext().DownloadURI(uri)
	dwnld.SetDestination("file://" + dlFile)
	var handle glib.SignalHandle
	handle, err = dwnld.Connect("finished", func() {
		defer os.RemoveAll(tmpDir)
		dwnld.HandlerDisconnect(handle)
		data, err := ioutil.ReadFile(dlFile)
		if err != nil {
			req.FinishError(err)
			return
		}
		var mimetype string
		resp, err := dwnld.GetResponse()
		if err != nil {
			mimetype = "application/octet-stream"
		} else {
			mimetype = resp.GetMimeType()
		}
		req.Finish(data, mimetype)
	})
	if err != nil {
		req.FinishError(err)
		os.RemoveAll(tmpDir)
		return
	}
}

// guessMimeFromExtension guesses a files mime type based on its file path.
func guessMimeFromExtension(path string) string {
	split := strings.Split(path, ".")
	// No extension to speak of, default to text.
	if len(split) == 1 {
		return "text/plain"
	}
	switch split[len(split)-1] {
	case "html":
		return "text/html"
	case "css":
		return "text/css"
	default:
		return "application/octet-stream"
	}
}
