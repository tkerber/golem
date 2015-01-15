package webkit

// #cgo pkg-config: webkit2gtk-4.0
// #include <webkit2/webkit2.h>
import "C"
import (
	"runtime"
	"unsafe"

	"github.com/conformal/gotk3/glib"
	"github.com/tkerber/golem/gtk"
)

// A ResponsePolicyDecision is a decision of whether or not to load a
// particular resource.
type ResponsePolicyDecision struct {
	*glib.Object
}

// native retrieves a precast native C pointer to the underlying gobject.
func (d *ResponsePolicyDecision) native() *C.WebKitResponsePolicyDecision {
	return (*C.WebKitResponsePolicyDecision)(unsafe.Pointer(d.Native()))
}

// Download downloads the resource referred to.
func (d *ResponsePolicyDecision) Download() {
	C.webkit_policy_decision_download(
		(*C.WebKitPolicyDecision)(unsafe.Pointer(d.Native())))
}

// Ignore ignores the resource referred to.
func (d *ResponsePolicyDecision) Ignore() {
	C.webkit_policy_decision_ignore(
		(*C.WebKitPolicyDecision)(unsafe.Pointer(d.Native())))
}

// Use proceeds to navigate to the resource referred to.
func (d *ResponsePolicyDecision) Use() {
	C.webkit_policy_decision_use(
		(*C.WebKitPolicyDecision)(unsafe.Pointer(d.Native())))
}

// GetResponse retrieves the uri response associated with this decision.
func (d *ResponsePolicyDecision) GetResponse() *URIResponse {
	cresp := C.webkit_response_policy_decision_get_response(d.native())
	resp := &URIResponse{&glib.Object{glib.ToGObject(unsafe.Pointer(cresp))}}
	resp.Object.RefSink()
	runtime.SetFinalizer(resp.Object, func(o *glib.Object) {
		gtk.GlibMainContextInvoke(o.Unref)
	})
	return resp
}
