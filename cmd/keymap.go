package cmd

// #cgo pkg-config: gdk-3.0
// #include <gdk/gdk.h>
// #include <stdlib.h>
import "C"
import "unsafe"

var colonKey = KeyvalFromName("colon")
var escapeKey = KeyvalFromName("Escape")
var returnKey = KeyvalFromName("Return")
var backSpaceKey = KeyvalFromName("BackSpace")

// KeyvalName gets the name of a key from it's key code.
func KeyvalName(keycode uint) string {
	cString := C.gdk_keyval_name(C.guint(keycode))
	if cString == nil {
		return "unknown"
	}
	return C.GoString((*C.char)(cString))
}

// KeyvalFromName gets the key code from the key's name.
func KeyvalFromName(name string) uint {
	cString := C.CString(name)
	defer C.free(unsafe.Pointer(cString))
	return uint(C.gdk_keyval_from_name((*C.gchar)(cString)))
}

// KeyvalToUnicode gets the unicode rune from a specified keyval.
func KeyvalToUnicode(keycode uint) rune {
	return rune(C.gdk_keyval_to_unicode(C.guint(keycode)))
}
