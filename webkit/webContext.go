package webkit

// #cgo pkg-config: webkit2gtk-4.0
// #include <stdlib.h>
// #include <webkit2/webkit2.h>
/*

extern void
cgoURISchemeRequestCallback(WebKitURISchemeRequest *req, gpointer f);

static inline void
go_webkit_web_context_register_uri_scheme(
		WebKitWebContext *c,
		gchar *scheme,
		gpointer callback) {
	webkit_web_context_register_uri_scheme(
		c,
		scheme,
		cgoURISchemeRequestCallback,
		callback,
		NULL);
}

*/
import "C"
import (
	"runtime"
	"unsafe"

	"github.com/conformal/gotk3/glib"
	"github.com/tkerber/golem/gtk"
)

const (
	// ProcessModelSharedSecondaryProcess specifies that the web process
	// should be shared between all WebViews.
	ProcessModelSharedSecondaryProcess = C.WEBKIT_PROCESS_MODEL_SHARED_SECONDARY_PROCESS
	// ProcessModelMultipleSecondaryProcesses specifies that (most) WebViews
	// should run their own web process.
	ProcessModelMultipleSecondaryProcesses = C.WEBKIT_PROCESS_MODEL_MULTIPLE_SECONDARY_PROCESSES
)

const (
	// CacheModelWebBrowser caches a large amount of previously visited
	// content.
	CacheModelWebBrowser = C.WEBKIT_CACHE_MODEL_WEB_BROWSER
	// CacheModelDocumentBrowser caches a moderate amount of content.
	CacheModelDocumentBrowser = C.WEBKIT_CACHE_MODEL_DOCUMENT_BROWSER
	// CacheModelDocumentViewer completely disables the cache.
	CacheModelDocumentViewer = C.WEBKIT_CACHE_MODEL_DOCUMENT_VIEWER
)

// The defaultWebContext is the WebContext which is used by default for new
// WebViews.
//
// May be nil until the default web context is retrieved.
var defaultWebContext *WebContext

// A WebContext is a wrapper around WebKitWebContext.
//
// It manages aspects common to all WebViews.
type WebContext struct {
	*glib.Object
	cookieManager   *CookieManager
	securityManager *SecurityManager
	uriSchemes      map[*func(*URISchemeRequest)]bool
}

// native retrieves the pre-cast pointer to the native C representation of the
// web context.
func (c *WebContext) native() *C.WebKitWebContext {
	return (*C.WebKitWebContext)(unsafe.Pointer(c.Native()))
}

// GetDefaultWebContext gets the default web context (i.e. the WebContext used
// by all WebViews by default).
func GetDefaultWebContext() *WebContext {
	if defaultWebContext == nil {
		wc := C.webkit_web_context_get_default()
		if wc == nil {
			panic("Failed to retrieve default web context.")
		}
		obj := &glib.Object{glib.ToGObject(unsafe.Pointer(wc))}
		obj.RefSink()
		runtime.SetFinalizer(obj, func(o *glib.Object) {
			gtk.GlibMainContextInvoke(o.Unref)
		})
		defaultWebContext = &WebContext{
			obj,
			nil,
			nil,
			make(map[*func(*URISchemeRequest)]bool, 5),
		}
	}
	return defaultWebContext
}

// SetWebExtensionsDirectory sets the directory in which web extensions can be
// found.
func (c *WebContext) SetWebExtensionsDirectory(to string) {
	cstr := C.CString(to)
	defer C.free(unsafe.Pointer(cstr))
	C.webkit_web_context_set_web_extensions_directory(
		c.native(),
		(*C.gchar)(cstr))
}

// SetProcessModel sets the model used for the distribution of web processes
// for WebViews.
//
// Should be one of ProcessModelSharedSecondaryProcess and
// ProcessModelMultipleSecondaryProcesses.
func (c *WebContext) SetProcessModel(to C.WebKitProcessModel) {
	C.webkit_web_context_set_process_model(c.native(), to)
}

// RegisterURIScheme registers a custom URI scheme.
func (c *WebContext) RegisterURIScheme(scheme string,
	callback func(*URISchemeRequest)) {

	// Prevents callback from being garbage collected until the WebContext
	// is destroyed.
	c.uriSchemes[&callback] = true
	cstr := C.CString(scheme)
	defer C.free(unsafe.Pointer(cstr))
	C.go_webkit_web_context_register_uri_scheme(
		c.native(),
		(*C.gchar)(cstr),
		C.gpointer(unsafe.Pointer(&callback)))
}

//export cgoURISchemeRequestCallback
func cgoURISchemeRequestCallback(req *C.WebKitURISchemeRequest, f C.gpointer) {
	goFunc := (*func(req *URISchemeRequest))(unsafe.Pointer(f))
	obj := &glib.Object{glib.ToGObject(unsafe.Pointer(req))}
	obj.RefSink()
	runtime.SetFinalizer(obj, func(o *glib.Object) {
		gtk.GlibMainContextInvoke(o.Unref)
	})
	goReq := &URISchemeRequest{obj}
	(*goFunc)(goReq)
}

// SetCacheModel sets the cache model to be used.
//
// Should be one of CacheModelWebBrowser, CacheModelDocumentViewer or
// CacheModelDocumentBrowser
func (c *WebContext) SetCacheModel(to C.WebKitCacheModel) {
	C.webkit_web_context_set_cache_model(c.native(), to)
}

// SetDiskCacheDirectory sets the directory of the cache on disk.
func (c *WebContext) SetDiskCacheDirectory(to string) {
	cStr := C.CString(to)
	defer C.free(unsafe.Pointer(cStr))
	C.webkit_web_context_set_disk_cache_directory(c.native(), (*C.gchar)(cStr))
}

// GetCookieManager retrieves the web context's cookie manager.
func (c *WebContext) GetCookieManager() *CookieManager {
	if c.cookieManager == nil {
		cptr := C.webkit_web_context_get_cookie_manager(c.native())
		c.cookieManager = wrapCookieManager(cptr)
	}
	return c.cookieManager
}

// GetSecurityManager retrieves the web context's security manager.
func (c *WebContext) GetSecurityManager() *SecurityManager {
	if c.securityManager == nil {
		cptr := C.webkit_web_context_get_security_manager(c.native())
		c.securityManager = wrapSecurityManager(cptr)
	}
	return c.securityManager
}

// DownloadURI requests the web context to download the specified URI.
func (c *WebContext) DownloadURI(uri string) *Download {
	cURI := C.CString(uri)
	defer C.free(unsafe.Pointer(cURI))
	cdl := C.webkit_web_context_download_uri(c.native(), (*C.gchar)(cURI))
	dl := &Download{&glib.Object{glib.ToGObject(unsafe.Pointer(cdl))}}
	dl.Object.RefSink()
	runtime.SetFinalizer(dl.Object, func(o *glib.Object) {
		gtk.GlibMainContextInvoke(o.Unref)
	})
	return dl
}
