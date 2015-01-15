package webkit

// #cgo pkg-config: webkit2gtk-4.0
// #include <webkit2/webkit2.h>
// #include <stdlib.h>
import "C"
import (
	"runtime"
	"time"
	"unsafe"

	"github.com/conformal/gotk3/glib"
	"github.com/tkerber/golem/gtk"
)

// init registers the Download type marshaler to glib.
func init() {
	glib.RegisterGValueMarshalers([]glib.TypeMarshaler{
		glib.TypeMarshaler{
			glib.Type(C.webkit_download_get_type()),
			func(ptr uintptr) (interface{}, error) {
				c := C.g_value_get_object((*C.GValue)(unsafe.Pointer(ptr)))
				obj := &glib.Object{glib.ToGObject(unsafe.Pointer(c))}
				d := &Download{obj}
				return d, nil
			},
		},
	})
}

// A Download is a wrapper around WebKitDownload
//
// It allows the tracking and controlling of downloads.
type Download struct {
	*glib.Object
}

// native returns a pre-cast native C pointer to the WebKitDownload object.
func (d *Download) native() *C.WebKitDownload {
	return (*C.WebKitDownload)(unsafe.Pointer(d.Native()))
}

// GetRequest gets the URIRequest associated with this download.
func (d *Download) GetRequest() *URIRequest {
	cReq := C.webkit_download_get_request(d.native())
	req := &URIRequest{&glib.Object{glib.ToGObject(unsafe.Pointer(cReq))}}
	req.Object.RefSink()
	runtime.SetFinalizer(req.Object, func(o *glib.Object) {
		gtk.GlibMainContextInvoke((*glib.Object).Unref, o)
	})
	return req
}

// GetDestination gets the target uri for the download.
func (d *Download) GetDestination() string {
	cstr := C.webkit_download_get_destination(d.native())
	return C.GoString((*C.char)(cstr))
}

// SetDestination sets the target uri for the download.
//
// Note the word uri - for most purposes you'll need to use the file:///
// protocol.
func (d *Download) SetDestination(to string) {
	cstr := C.CString(to)
	defer C.free(unsafe.Pointer(cstr))
	C.webkit_download_set_destination(d.native(), (*C.gchar)(cstr))
}

// Cancel cancells the download.
func (d *Download) Cancel() {
	C.webkit_download_cancel(d.native())
}

// GetEstimatedProgress gets an estimate for the download progress.
//
// This is given as a fraction, i.e. a float between 0 and 1.
func (d *Download) GetEstimatedProgress() float64 {
	return float64(C.webkit_download_get_estimated_progress(d.native()))
}

// GetElapsedTime gets the Duration which has passed since the download
// started.
func (d *Download) GetElapsedTime() time.Duration {
	seconds := float64(C.webkit_download_get_elapsed_time(d.native()))
	return time.Duration(float64(time.Second) * seconds)
}

// GetReceivedDataLength gets the length of the data received in bytes.
func (d *Download) GetReceivedDataLength() int64 {
	return int64(C.webkit_download_get_received_data_length(d.native()))
}

// GetWebView gets the web view associated with this download.
func (d *Download) GetWebView() (*WebView, error) {
	cWv := C.webkit_download_get_web_view(d.native())
	if cWv == nil {
		return nil, errNilPtr
	}
	obj := &glib.Object{glib.ToGObject(unsafe.Pointer(cWv))}
	obj.Ref()
	runtime.SetFinalizer(obj, func(o *glib.Object) {
		gtk.GlibMainContextInvoke((*glib.Object).Unref, o)
	})
	return wrapWebView(obj), nil
}

// GetResponse gets the uri response to the download request.
//
// Returns an error if the response is not yet available.
func (d *Download) GetResponse() (*URIResponse, error) {
	cresp := C.webkit_download_get_response(d.native())
	if cresp == nil {
		return nil, errNilPtr
	}
	resp := &URIResponse{&glib.Object{glib.ToGObject(unsafe.Pointer(cresp))}}
	resp.Object.Ref()
	runtime.SetFinalizer(resp.Object, (*glib.Object).Unref)
	return resp, nil
}
