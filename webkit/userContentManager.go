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

// A UserContentManager is a wrapper around WebKitUserContentManager.
//
// It manages user content (i.e. scripts and style sheets), for WebViews which
// link to it.
type UserContentManager struct {
	*glib.Object
}

// NewUserContentManager creates a new (blank) UserContentManager.
func NewUserContentManager() (*UserContentManager, error) {
	ucm := C.webkit_user_content_manager_new()
	if ucm == nil {
		return nil, errNilPtr
	}
	obj := &glib.Object{glib.ToGObject(unsafe.Pointer(ucm))}
	obj.RefSink()
	runtime.SetFinalizer(obj, func(o *glib.Object) {
		gtk.GlibMainContextInvoke(o.Unref)
	})
	return &UserContentManager{obj}, nil
}

// AddStyleSheet attaches a UserStyleSheet to this UserContentManager.
func (ucm *UserContentManager) AddStyleSheet(s *UserStyleSheet) {
	C.webkit_user_content_manager_add_style_sheet(
		(*C.WebKitUserContentManager)(unsafe.Pointer(ucm.Native())),
		(*C.WebKitUserStyleSheet)(unsafe.Pointer(s.Native())))
}
