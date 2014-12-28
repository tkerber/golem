package main

// #cgo pkg-config: webkit2gtk-4.0
// #include <webkit2/webkit2.h>
// #include <stdlib.h>
import "C"
import (
	"go/build"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/tkerber/golem/webkit"
)

// webkitInit initializes webkit for golem's use.
func (g *golem) webkitInit() {
	// TODO figure out a better way to reference this. (i.e. without the source)
	extenPath := ""
	for _, src := range build.Default.SrcDirs() {
		p := filepath.Join(src, "github.com", "tkerber", "golem", "web_extension")
		if _, err := os.Stat(p); err == nil {
			extenPath = p
			break
		}
	}
	if extenPath == "" {
		panic("Failed to find source files!")
	}

	c := webkit.GetDefaultWebContext()
	c.SetWebExtensionsDirectory(extenPath)

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

	// TODO this is temporary.
	c.RegisterURIScheme("golem", golemSchemeHandler)
}

// golemSchemeHandler handles request to the 'golem:' scheme.
func golemSchemeHandler(req *webkit.URISchemeRequest) {
	req.Finish([]byte("<html><head><title>Golem</title></head><body><h1>Golem Home Page</h1><p>And stuff.</p></body></html>"), "text/html")
}
