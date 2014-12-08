package webkit

// #cgo pkg-config: webkit2gtk-4.0
// #include <webkit2/webkit2.h>
// #include <stdlib.h>
import "C"
import (
	"runtime"
	"unsafe"
)

const (
	UserContentInjectAllFrames = C.WEBKIT_USER_CONTENT_INJECT_ALL_FRAMES
	UserContentInjectTopFrame  = C.WEBKIT_USER_CONTENT_INJECT_TOP_FRAME
)

const (
	UserStyleLevelUser   = C.WEBKIT_USER_STYLE_LEVEL_USER
	UserStyleLevelAuthor = C.WEBKIT_USER_STYLE_LEVEL_AUTHOR
)

type UserStyleSheet struct {
	native uintptr
}

func NewUserStyleSheet(
	source string,
	frames C.WebKitUserContentInjectedFrames,
	level C.WebKitUserStyleLevel,
	whitelist []string,
	blacklist []string) (*UserStyleSheet, error) {

	csrc := C.CString(source)
	defer C.free(unsafe.Pointer(csrc))
	cwl := make([]*C.gchar, len(whitelist)+1)
	cbl := make([]*C.gchar, len(whitelist)+1)
	for i, item := range whitelist {
		cstr := C.CString(item)
		defer C.free(unsafe.Pointer(cstr))
		cwl[i] = (*C.gchar)(cstr)
	}
	cwl[len(whitelist)] = nil
	for i, item := range blacklist {
		cstr := C.CString(item)
		defer C.free(unsafe.Pointer(cstr))
		cbl[i] = (*C.gchar)(cstr)
	}
	cbl[len(whitelist)] = nil
	ccss := C.webkit_user_style_sheet_new(
		(*C.gchar)(csrc),
		frames,
		level,
		&cwl[0],
		&cbl[0])
	if ccss == nil {
		return nil, errNilPtr
	}
	css := &UserStyleSheet{uintptr(unsafe.Pointer(ccss))}
	runtime.SetFinalizer(css, (*UserStyleSheet).unref)
	return css, nil
}

func (css *UserStyleSheet) unref() {
	C.webkit_user_style_sheet_unref(
		(*C.WebKitUserStyleSheet)(unsafe.Pointer(css.native)))
}

func (css *UserStyleSheet) Native() uintptr {
	return css.native
}
