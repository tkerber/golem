package webkit

// #cgo pkg-config: webkit2gtk-4.0
// #include <webkit2/webkit2.h>
import "C"
import (
	"runtime"
	"unsafe"

	"github.com/conformal/gotk3/glib"
)

// A NavigationPolicyDecision is a decision of whether or not to navigate to
// a page.
type NavigationPolicyDecision struct {
	*glib.Object
}

// native returns a precast native C pointer to the underlying gobject.
func (d *NavigationPolicyDecision) native() *C.WebKitNavigationPolicyDecision {
	return (*C.WebKitNavigationPolicyDecision)(unsafe.Pointer(d.Native()))
}

// Download downloads the resource referred to.
func (d *NavigationPolicyDecision) Download() {
	C.webkit_policy_decision_download(
		(*C.WebKitPolicyDecision)(unsafe.Pointer(d.Native())))
}

// Ignore ignores the resource referred to.
func (d *NavigationPolicyDecision) Ignore() {
	C.webkit_policy_decision_ignore(
		(*C.WebKitPolicyDecision)(unsafe.Pointer(d.Native())))
}

// Use proceeds to navigate to the resource referred to.
func (d *NavigationPolicyDecision) Use() {
	C.webkit_policy_decision_use(
		(*C.WebKitPolicyDecision)(unsafe.Pointer(d.Native())))
}

// GetNavigationAction returns the navigation action causing the navigation.
func (d *NavigationPolicyDecision) GetNavigationAction() *NavigationAction {
	cnav := C.webkit_navigation_policy_decision_get_navigation_action(
		d.native())
	cnav = C.webkit_navigation_action_copy(cnav)
	nav := &NavigationAction{cnav}
	runtime.SetFinalizer(nav, (*NavigationAction).free)
	return nav
}
