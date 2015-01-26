package webkit

// #cgo pkg-config: webkit2gtk-4.0
// #include <webkit2/webkit2.h>
// #include <stdlib.h>
import "C"
import (
	"unsafe"

	"github.com/conformal/gotk3/glib"
)

const (
	FindOptionsNone                          = C.WEBKIT_FIND_OPTIONS_NONE
	FindOptionsCaseInsensitive               = C.WEBKIT_FIND_OPTIONS_CASE_INSENSITIVE
	FindOptionsAtWordStarts                  = C.WEBKIT_FIND_OPTIONS_AT_WORD_STARTS
	FindOptionsTreatMedialCapitalAsWordStart = C.WEBKIT_FIND_OPTIONS_TREAT_MEDIAL_CAPITAL_AS_WORD_START
	FindOptionsBackwards                     = C.WEBKIT_FIND_OPTIONS_BACKWARDS
	FindOptionsWrapAround                    = C.WEBKIT_FIND_OPTIONS_WRAP_AROUND
)

type FindController struct {
	*glib.Object
}

func (fc *FindController) native() *C.WebKitFindController {
	return (*C.WebKitFindController)(unsafe.Pointer(fc.Native()))
}

func (fc *FindController) Search(term string, options C.guint32) {
	cstr := C.CString(term)
	defer C.free(unsafe.Pointer(cstr))
	C.webkit_find_controller_search(
		fc.native(),
		(*C.gchar)(cstr),
		options,
		C.G_MAXUINT)
}

func (fc *FindController) SearchFinish() {
	C.webkit_find_controller_search_finish(fc.native())
}

func (fc *FindController) SearchNext() {
	C.webkit_find_controller_search_next(fc.native())
}

func (fc *FindController) SearchPrevious() {
	C.webkit_find_controller_search_previous(fc.native())
}

func (fc *FindController) GetSearchText() string {
	cstr := C.webkit_find_controller_get_search_text(fc.native())
	return C.GoString((*C.char)(cstr))
}

func (fc *FindController) GetOptions() C.guint32 {
	return C.webkit_find_controller_get_options(fc.native())
}
