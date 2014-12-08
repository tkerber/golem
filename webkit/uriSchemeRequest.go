package webkit

// #cgo pkg-config: webkit2gtk-4.0
// #include <stdlib.h>
// #include <webkit2/webkit2.h>
import "C"
import (
	"unsafe"
)

type URISchemeRequest struct {
	native uintptr
}

func (r *URISchemeRequest) GetScheme() string {
	cstr := C.webkit_uri_scheme_request_get_scheme(
		(*C.WebKitURISchemeRequest)(unsafe.Pointer(r.native)))
	return C.GoString((*C.char)(cstr))
}

func (r *URISchemeRequest) GetURI() string {
	cstr := C.webkit_uri_scheme_request_get_uri(
		(*C.WebKitURISchemeRequest)(unsafe.Pointer(r.native)))
	return C.GoString((*C.char)(cstr))
}

func (r *URISchemeRequest) GetPath() string {
	cstr := C.webkit_uri_scheme_request_get_path(
		(*C.WebKitURISchemeRequest)(unsafe.Pointer(r.native)))
	return C.GoString((*C.char)(cstr))
}

func (r *URISchemeRequest) Finish(data []byte, mimeType string) {
	// TODO: use a reader instead of data; and think of some way to transform it
	// into a GInputStream.
	cstr := C.CString(mimeType)
	defer C.free(unsafe.Pointer(cstr))
	s := C.g_memory_input_stream_new_from_data(
		unsafe.Pointer(&data[0]),
		C.gssize(len(data)),
		nil)
	C.webkit_uri_scheme_request_finish(
		(*C.WebKitURISchemeRequest)(unsafe.Pointer(r.native)),
		s,
		C.gint64(len(data)),
		(*C.gchar)(cstr))
	C.g_object_unref(C.gpointer(s))
}
