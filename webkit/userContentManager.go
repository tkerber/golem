package webkit

// #cgo pkg-config: webkit2gtk-4.0
// #include <webkit2/webkit2.h>
import "C"
import (
	"runtime"
	"unsafe"

	"github.com/conformal/gotk3/glib"
)

type UserContentManager struct {
	*glib.Object
}

func NewUserContentManager() (*UserContentManager, error) {
	ucm := C.webkit_user_content_manager_new()
	if ucm == nil {
		return nil, errNilPtr
	}
	obj := &glib.Object{glib.ToGObject(unsafe.Pointer(ucm))}
	obj.RefSink()
	runtime.SetFinalizer(obj, (*glib.Object).Unref)
	return &UserContentManager{obj}, nil
}

func (ucm *UserContentManager) AddStyleSheet(s *UserStyleSheet) {
	C.webkit_user_content_manager_add_style_sheet(
		(*C.WebKitUserContentManager)(unsafe.Pointer(ucm.Native())),
		(*C.WebKitUserStyleSheet)(unsafe.Pointer(s.Native())))
}
