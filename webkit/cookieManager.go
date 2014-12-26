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

const (
	// CookiePeristentStorageText specifies that the cookies should be saved
	// in a text format.
	CookiePersistentStorageText = C.WEBKIT_COOKIE_PERSISTENT_STORAGE_TEXT
	// CookiePersistentStorageSQLite specifies that the cookies should be
	// saves in a SQLite format.
	CookiePersistentStorageSQLite = C.WEBKIT_COOKIE_PERSISTENT_STORAGE_SQLITE
)

// CookieManager wraps WebKitCookieManager.
//
// It keeps track or cookie policy.
type CookieManager struct {
	*glib.Object
}

// wrapCookieManager wraps a native C WebKitCookieManager into a go struct.
func wrapCookieManager(cptr *C.WebKitCookieManager) *CookieManager {
	obj := &glib.Object{glib.ToGObject(unsafe.Pointer(cptr))}
	obj.RefSink()
	runtime.SetFinalizer(obj, (*glib.Object).Unref)
	return &CookieManager{obj}
}

// native retrieves the pre-cast native C WebKitCookieManager.
func (cm *CookieManager) native() *C.WebKitCookieManager {
	return (*C.WebKitCookieManager)(unsafe.Pointer(cm.Native()))
}

// SetPersistentStorage sets the file and format in which to store cookies
// persistently.
func (cm *CookieManager) SetPersistentStorage(
	file string,
	storage C.WebKitCookiePersistentStorage) {

	cStr := C.CString(file)
	defer C.free(unsafe.Pointer(cStr))
	C.webkit_cookie_manager_set_persistent_storage(
		cm.native(),
		(*C.gchar)(cStr),
		storage)
}
