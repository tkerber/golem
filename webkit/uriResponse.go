package webkit

// #cgo pkg-config: webkit2gtk-4.0
// #include <webkit2/webkit2.h>
import "C"
import (
	"unsafe"

	"github.com/conformal/gotk3/glib"
)

// A URIResponse captures basic information about the response to a web
// request.
type URIResponse struct {
	*glib.Object
}

// native returns a pre-cast native C pointer to the gobject.
func (r *URIResponse) native() *C.WebKitURIResponse {
	return (*C.WebKitURIResponse)(unsafe.Pointer(r.Native()))
}

// GetMimeType retrieves the mime type of the response.
func (r *URIResponse) GetMimeType() string {
	cstr := C.webkit_uri_response_get_mime_type(r.native())
	return C.GoString((*C.char)(cstr))
}

// GetURI gets the uri for which this is the response.
func (r *URIResponse) GetURI() string {
	cstr := C.webkit_uri_response_get_uri(r.native())
	return C.GoString((*C.char)(cstr))
}
