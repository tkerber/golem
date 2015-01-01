package webkit

// #cgo pkg-config: webkit2gtk-4.0
// #include <webkit2/webkit2.h>
import "C"
import (
	"unsafe"

	"github.com/conformal/gotk3/glib"
)

// A UriResponse captures basic information about the response to a web
// request.
type UriResponse struct {
	*glib.Object
}

// native returns a pre-cast native C pointer to the gobject.
func (r *UriResponse) native() *C.WebKitURIResponse {
	return (*C.WebKitURIResponse)(unsafe.Pointer(r.Native()))
}

// GetMimeType retrieves the mime type of the response.
func (r *UriResponse) GetMimeType() string {
	cstr := C.webkit_uri_response_get_mime_type(r.native())
	return C.GoString((*C.char)(cstr))
}

// GetUri gets the uri for which this is the response.
func (r *UriResponse) GetUri() string {
	cstr := C.webkit_uri_response_get_uri(r.native())
	return C.GoString((*C.char)(cstr))
}
