package webkit

// #cgo pkg-config: webkit2gtk-4.0
// #include <webkit2/webkit2.h>
import "C"
import (
	"runtime"
	"unsafe"

	"github.com/conformal/gotk3/glib"
)

// A BackForwardList is a list of elements to go back or forward in browser
// history to.
type BackForwardList struct {
	*glib.Object
}

// native returns the native C representation of the BFL.
func (bfl *BackForwardList) native() *C.WebKitBackForwardList {
	return (*C.WebKitBackForwardList)(unsafe.Pointer(bfl.Native()))
}

// GetNthItem retrieves the nth item, relative to the current item, from the
// BFL.
func (bfl *BackForwardList) GetNthItem(n int) (*BackForwardListItem, bool) {
	cobj := C.webkit_back_forward_list_get_nth_item(bfl.native(), (C.gint)(n))
	if cobj == nil {
		return nil, false
	}
	obj := &glib.Object{glib.ToGObject(unsafe.Pointer(cobj))}
	obj.RefSink()
	runtime.SetFinalizer(obj, (*glib.Object).Unref)
	return &BackForwardListItem{glib.InitiallyUnowned{obj}}, true
}

// GetNthItemWeak tries to behave like GetNthItem, but if it fails, tries to
// find the closest index for which it doesn't.
//
// If there are no specified items in the given direction (or n == 0), false
// is returned.
//
// Yes this isn't strictly a wrapper function, but it's very useful.
func (bfl *BackForwardList) GetNthItemWeak(n int) (*BackForwardListItem, bool) {
	var list *C.GList
	back := false
	if n < 0 {
		n = -n
		back = true
		list = C.webkit_back_forward_list_get_back_list(bfl.native())
	} else if n > 0 {
		list = C.webkit_back_forward_list_get_forward_list(bfl.native())
	} else {
		return nil, false
	}
	defer C.g_list_free(list)
	length := C.g_list_length(list)
	if length == 0 {
		return nil, false
	}
	if n > int(length) {
		n = int(length)
	}
	var i C.guint
	if back {
		i = C.guint(n - 1)
	} else {
		i = length - C.guint(n)
	}
	ptr := C.g_list_nth_data(list, i)
	obj := &glib.Object{glib.ToGObject(unsafe.Pointer(ptr))}
	obj.RefSink()
	runtime.SetFinalizer(obj, (*glib.Object).Unref)
	return &BackForwardListItem{glib.InitiallyUnowned{obj}}, true
}

// A BackForwardListItem is a single item in a BackForwardList.
type BackForwardListItem struct {
	glib.InitiallyUnowned
}
