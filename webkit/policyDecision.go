package webkit

// #cgo pkg-config: webkit2gtk-4.0
// #include <webkit2/webkit2.h>
/*
static inline gboolean
cgo_is_response_policy_decision(gpointer obj) {
	return WEBKIT_IS_RESPONSE_POLICY_DECISION(obj);
}

static inline gboolean
cgo_is_navigation_policy_decision(gpointer obj) {
	return WEBKIT_IS_NAVIGATION_POLICY_DECISION(obj);
}
*/
import "C"
import (
	"unsafe"

	"github.com/conformal/gotk3/glib"
)

// init registers the PolicyDecision type marshaler to glib.
func init() {
	glib.RegisterGValueMarshalers([]glib.TypeMarshaler{
		glib.TypeMarshaler{
			glib.Type(C.webkit_policy_decision_get_type()),
			func(ptr uintptr) (interface{}, error) {
				c := C.g_value_get_object((*C.GValue)(unsafe.Pointer(ptr)))
				obj := &glib.Object{glib.ToGObject(unsafe.Pointer(c))}
				var d PolicyDecision
				if gobool(C.cgo_is_response_policy_decision(c)) {
					d = &ResponsePolicyDecision{obj}
				} else if gobool(C.cgo_is_navigation_policy_decision(c)) {
					d = &NavigationPolicyDecision{obj}
				} else {
					panic("unknown decision type!")
				}
				return d, nil
			},
		},
	})
}

// A PolicyDecision is an interface implemented by the various types of
// policy decisions, and has methods used to make said decisions.
type PolicyDecision interface {
	// Download downloads the resource referred to.
	Download()
	// Ignore makes the decision to take no action.
	Ignore()
	// Use makes the decision to go ahead with the suggested action.
	Use()
}
