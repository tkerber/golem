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
	// UserContentInjectAllFrames specifies that user content should be
	// injected into all frames.
	UserContentInjectAllFrames = C.WEBKIT_USER_CONTENT_INJECT_ALL_FRAMES
	// UserContentInjectTopFrame specifies that user content should be
	// injected only into the top-level frame.
	UserContentInjectTopFrame = C.WEBKIT_USER_CONTENT_INJECT_TOP_FRAME
)

const (
	// UserStyleLevelUser specifies that the style is considered dictated by
	// the user, and overrides any other conflicting CSS.
	UserStyleLevelUser = C.WEBKIT_USER_STYLE_LEVEL_USER
	// UserStyleLevelAuthor specifies that the style is considered dictated by
	// the website, and may be overriden by conflicting CSS.
	UserStyleLevelAuthor = C.WEBKIT_USER_STYLE_LEVEL_AUTHOR
)

// A UserStyleSheet is a wrapper around WebKitUserStyleSheet.
//
// It represents user-defined CSS rules, which can be applied to a website.
type UserStyleSheet struct {
	native uintptr
}

// NewUserStyleSheet creates a new user style sheet.
//
// It takes the raw CSS source code, one of UserContentInjectAllFrames or
// UserContentInjectTopFrame; one of UserStyleLevelUser or
// UserStyleLevelAuthor; and uri white- and blacklists.
//
// Passing nil for the whitelist implies all URIs are whitelisted; passing
// nil for the blacklist implies no URIs are blacklisted.
func NewUserStyleSheet(
	source string,
	frames C.WebKitUserContentInjectedFrames,
	level C.WebKitUserStyleLevel,
	whitelist []string,
	blacklist []string) (*UserStyleSheet, error) {

	csrc := C.CString(source)
	defer C.free(unsafe.Pointer(csrc))

	// Creates C whitelist, NULL terminated and w/ c strings.
	cwl := make([]*C.gchar, len(whitelist)+1)
	for i, item := range whitelist {
		cstr := C.CString(item)
		defer C.free(unsafe.Pointer(cstr))
		cwl[i] = (*C.gchar)(cstr)
	}
	cwl[len(whitelist)] = nil

	// Slight repetition here, but extracting into a method would prevent
	// using defer.
	// Creates C blaclist, NULL terminated and w/ c strings.
	cbl := make([]*C.gchar, len(blacklist)+1)
	for i, item := range blacklist {
		cstr := C.CString(item)
		defer C.free(unsafe.Pointer(cstr))
		cbl[i] = (*C.gchar)(cstr)
	}
	cbl[len(blacklist)] = nil

	// Actually create the CSS.
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
	// It starts with a ref count of 1, so no need to manually ref it.
	runtime.SetFinalizer(css, (*UserStyleSheet).unref)
	return css, nil
}

// unref dereferences the style sheet.
func (css *UserStyleSheet) unref() {
	C.webkit_user_style_sheet_unref(
		(*C.WebKitUserStyleSheet)(unsafe.Pointer(css.native)))
}

// Native returns the pointer to the native C object of the style sheet.
func (css *UserStyleSheet) Native() uintptr {
	return css.native
}
