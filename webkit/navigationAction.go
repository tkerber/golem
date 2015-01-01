package webkit

// #cgo pkg-config: webkit2gtk-4.0
// #include <webkit2/webkit2.h>
import "C"
import (
	"runtime"
	"unsafe"

	"github.com/conformal/gotk3/glib"
)

// A NavigationAction describes the action taken to cause a navigation request.
type NavigationAction struct {
	native *C.WebKitNavigationAction
}

// free frees the internal C data.
func (a *NavigationAction) free() {
	C.webkit_navigation_action_free(a.native)
}

// GetRequest gets the request associated with the navigation action.
func (a *NavigationAction) GetRequest() *UriRequest {
	creq := C.webkit_navigation_action_get_request(a.native)
	req := &UriRequest{&glib.Object{glib.ToGObject(unsafe.Pointer(creq))}}
	req.Object.RefSink()
	runtime.SetFinalizer(req.Object, (*glib.Object).Unref)
	return req
}

// GetMouseButton returns the mouse button used to trigger the action, or 0
// if it wasn't triggered by mouse.
func (a *NavigationAction) GetMouseButton() uint {
	return uint(C.webkit_navigation_action_get_mouse_button(a.native))
}

// GetModifiers returns the modifier key mask pressed when the action was made.
func (a *NavigationAction) GetModifiers() uint {
	return uint(C.webkit_navigation_action_get_modifiers(a.native))
}
