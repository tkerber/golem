package webkit

// #cgo pkg-config: webkit2gtk-4.0
// #include <webkit2/webkit2.h>
// #include <stdlib.h>
import "C"
import (
	"runtime"
	"unsafe"

	"github.com/conformal/gotk3/glib"
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
	runtime.SetFinalizer(obj, (*glib.Object).Unref)
	return &SecurityManager{obj}
}

// native returns a pre-cast native C WebKitSecurityManager.
func (sm *SecurityManager) native() *C.WebKitSecurityManager {
	return (*C.WebKitSecurityManager)(unsafe.Pointer(sm.Native()))
}

// RegisterUriSchemeAsLocal registers a uri scheme as being local, i.e. non-
// local pages can't interact with it.
func (sm *SecurityManager) RegisterUriSchemeAsLocal(scheme string) {
	cstr := (*C.gchar)(C.CString(scheme))
	defer C.free(unsafe.Pointer(cstr))
	C.webkit_security_manager_register_uri_scheme_as_local(sm.native(), cstr)
}

// RegisterUriSchemeAsNoAccess registers a uri scheme to have no access to
// web pages in other schemes.
func (sm *SecurityManager) RegisterUriSchemeAsNoAccess(scheme string) {
	cstr := (*C.gchar)(C.CString(scheme))
	defer C.free(unsafe.Pointer(cstr))
	C.webkit_security_manager_register_uri_scheme_as_no_access(sm.native(), cstr)
}

// RegisterUriSchemeAsDisplayIsolated registers a uri scheme such that only
// pages on the same scheme can display it.
func (sm *SecurityManager) RegisterUriSchemeAsDisplayIsolated(scheme string) {
	cstr := (*C.gchar)(C.CString(scheme))
	defer C.free(unsafe.Pointer(cstr))
	C.webkit_security_manager_register_uri_scheme_as_display_isolated(sm.native(), cstr)
}

// RegisterUriSchemeAsSecure silences mixed content warning for this scheme on
// https sites.
func (sm *SecurityManager) RegisterUriSchemeAsSecure(scheme string) {
	cstr := (*C.gchar)(C.CString(scheme))
	defer C.free(unsafe.Pointer(cstr))
	C.webkit_security_manager_register_uri_scheme_as_secure(sm.native(), cstr)
}

// RegisterUriSchemeAsCorsEnabled registers the scheme as a CORS enabled scheme.
func (sm *SecurityManager) RegisterUriSchemeAsCorsEnabled(scheme string) {
	cstr := (*C.gchar)(C.CString(scheme))
	defer C.free(unsafe.Pointer(cstr))
	C.webkit_security_manager_register_uri_scheme_as_cors_enabled(sm.native(), cstr)
}

// RegisterUriSchemeAsEmptyDocuments registers a uri scheme as an empty document
// scheme.
func (sm *SecurityManager) RegisterUriSchemeAsEmptyDocument(scheme string) {
	cstr := (*C.gchar)(C.CString(scheme))
	defer C.free(unsafe.Pointer(cstr))
	C.webkit_security_manager_register_uri_scheme_as_empty_document(sm.native(), cstr)
}
