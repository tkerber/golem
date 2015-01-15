package webkit

// #cgo pkg-config: webkit2gtk-4.0
// #include <webkit2/webkit2.h>
// #include <stdlib.h>
import "C"
import (
	"runtime"
	"unsafe"

	"github.com/conformal/gotk3/glib"
	"github.com/tkerber/golem/gtk"
)

// A SecurityManager is a wrapper around WebKitSecurityManager.
//
// It manages the security level of different URI schemes.
type SecurityManager struct {
	*glib.Object
}

// wrapSecurityManager converts a native C WebKitSecurityManager into its go
// wrapper.
func wrapSecurityManager(cptr *C.WebKitSecurityManager) *SecurityManager {
	obj := &glib.Object{glib.ToGObject(unsafe.Pointer(cptr))}
	obj.RefSink()
	runtime.SetFinalizer(obj, func(o *glib.Object) {
		gtk.GlibMainContextInvoke(o.Unref)
	})
	return &SecurityManager{obj}
}

// native returns a pre-cast native C WebKitSecurityManager.
func (sm *SecurityManager) native() *C.WebKitSecurityManager {
	return (*C.WebKitSecurityManager)(unsafe.Pointer(sm.Native()))
}

// RegisterURISchemeAsLocal registers a uri scheme as being local, i.e. non-
// local pages can't interact with it.
func (sm *SecurityManager) RegisterURISchemeAsLocal(scheme string) {
	cstr := (*C.gchar)(C.CString(scheme))
	defer C.free(unsafe.Pointer(cstr))
	C.webkit_security_manager_register_uri_scheme_as_local(sm.native(), cstr)
}

// RegisterURISchemeAsNoAccess registers a uri scheme to have no access to
// web pages in other schemes.
func (sm *SecurityManager) RegisterURISchemeAsNoAccess(scheme string) {
	cstr := (*C.gchar)(C.CString(scheme))
	defer C.free(unsafe.Pointer(cstr))
	C.webkit_security_manager_register_uri_scheme_as_no_access(
		sm.native(),
		cstr)
}

// RegisterURISchemeAsDisplayIsolated registers a uri scheme such that only
// pages on the same scheme can display it.
func (sm *SecurityManager) RegisterURISchemeAsDisplayIsolated(scheme string) {
	cstr := (*C.gchar)(C.CString(scheme))
	defer C.free(unsafe.Pointer(cstr))
	C.webkit_security_manager_register_uri_scheme_as_display_isolated(
		sm.native(),
		cstr)
}

// RegisterURISchemeAsSecure silences mixed content warning for this scheme on
// https sites.
func (sm *SecurityManager) RegisterURISchemeAsSecure(scheme string) {
	cstr := (*C.gchar)(C.CString(scheme))
	defer C.free(unsafe.Pointer(cstr))
	C.webkit_security_manager_register_uri_scheme_as_secure(
		sm.native(),
		cstr)
}

// RegisterURISchemeAsCorsEnabled registers the scheme as a CORS enabled scheme.
func (sm *SecurityManager) RegisterURISchemeAsCorsEnabled(scheme string) {
	cstr := (*C.gchar)(C.CString(scheme))
	defer C.free(unsafe.Pointer(cstr))
	C.webkit_security_manager_register_uri_scheme_as_cors_enabled(
		sm.native(),
		cstr)
}

// RegisterURISchemeAsEmptyDocument registers a uri scheme as an empty document
// scheme.
func (sm *SecurityManager) RegisterURISchemeAsEmptyDocument(scheme string) {
	cstr := (*C.gchar)(C.CString(scheme))
	defer C.free(unsafe.Pointer(cstr))
	C.webkit_security_manager_register_uri_scheme_as_empty_document(
		sm.native(),
		cstr)
}
