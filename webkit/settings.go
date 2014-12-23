package webkit

// #cgo pkg-config: webkit2gtk-4.0
// #include <webkit2/webkit2.h>
// #include <stdlib.h>
/*
void
_g_object_set_one(GObject *obj, gchar *prop, void *ptr) {
	g_object_set(obj, prop, *(gpointer **)ptr, NULL);
}

void
_g_object_get_one(GObject *obj, gchar *prop, void *ptr) {
	g_object_get(obj, prop, ptr, NULL);
}
*/
import "C"
import (
	"fmt"
	"reflect"
	"runtime"
	"unsafe"

	"github.com/conformal/gotk3/glib"
)

var (
	// All boolean settings
	boolSetting = map[string]bool{
		"allow-modal-dialogs":                       true,
		"auto-load-images":                          true,
		"draw-compositing-indicators":               true,
		"enable-accelerated-2d-canvas":              true,
		"enable-caret-browsing":                     true,
		"enable-developer-extras":                   true,
		"enable-dns-prefetching":                    true,
		"enable-frame-flattening":                   true,
		"enable-fullscreen":                         true,
		"enable-html5-database":                     true,
		"enable-html5-local-storage":                true,
		"enable-hyperlink-auditing":                 true,
		"enable-java":                               true,
		"enable-javascript":                         true,
		"enable-media-stream":                       true,
		"enable-mediasource":                        true,
		"enable-offline-web-application-cache":      true,
		"enable-page-cache":                         true,
		"enable-plugins":                            true,
		"enable-private-browsing":                   true,
		"enable-resizable-text-areas":               true,
		"enable-site-specific-quirks":               true,
		"enable-smooth-scrolling":                   true,
		"enable-spatial-navigation":                 true,
		"enable-tabs-to-links":                      true,
		"enable-webaudio":                           true,
		"enable-webgl":                              true,
		"enable-write-console-messages-to-stdout":   true,
		"enable-xss-auditor":                        true,
		"javascript-can-access-clipboard":           true,
		"javascript-can-open-windows-automatically": true,
		"load-icons-ignoring-image-load-setting":    true,
		"media-playback-allows-inline":              true,
		"media-playback-requires-user-gesture":      true,
		"print-backgrounds":                         true,
		"zoom-text-only":                            true,
	}
	// All string settings
	stringSetting = map[string]bool{
		"cursive-font-family":    true,
		"default-charset":        true,
		"default-font-family":    true,
		"fantasy-font-family":    true,
		"monospace-font-family":  true,
		"pictograph-font-family": true,
		"sans-serif-font-family": true,
		"serif-font-family":      true,
		"user-agent":             true,
	}
	// All uint settings
	uintSetting = map[string]bool{
		"default-font-size":           true,
		"default-monospace-font-size": true,
		"minimum-font-size":           true,
	}
)

// Settings are a collection of settings for webkit.
type Settings struct {
	*glib.Object
}

var typeUint = reflect.TypeOf(uint(0))
var typeString = reflect.TypeOf("")
var typeBool = reflect.TypeOf(false)

// NewSettings creates a new set of settings, with the default contents.
func NewSettings() *Settings {
	s := C.webkit_settings_new()
	return wrapSettings(s)
}

// wrapSettings converts settings from their C representation to their go
// representation.
func wrapSettings(ptr *C.WebKitSettings) *Settings {
	obj := &glib.Object{glib.ToGObject(unsafe.Pointer(ptr))}
	obj.RefSink()
	runtime.SetFinalizer(obj, (*glib.Object).Unref)
	return &Settings{obj}
}

// Clone clones a set of settings.
func (s *Settings) Clone() *Settings {
	sNew := NewSettings()
	for setting := range boolSetting {
		sNew.SetBool(setting, s.GetBool(setting))
	}
	for setting := range stringSetting {
		sNew.SetString(setting, s.GetString(setting))
	}
	for setting := range uintSetting {
		sNew.SetUint(setting, s.GetUint(setting))
	}
	return sNew
}

// object converts settings from their go representation to their C gobject
// representation.
func (s *Settings) object() *C.GObject {
	return (*C.GObject)(unsafe.Pointer(s.Native()))
}

// SetBool sets a single boolean setting.
//
// Type checks are not made, if the setting isn't boolean, this method will
// panic.
func (s *Settings) SetBool(setting string, to bool) {
	cStr := C.CString(setting)
	defer C.free(unsafe.Pointer(cStr))
	cBool := cbool(to)
	C._g_object_set_one(
		s.object(),
		(*C.gchar)(cStr),
		unsafe.Pointer(&cBool))
}

// SetString sets a single string settings.
//
// Type checks are not made, if the setting isn't a string, this method will
// panic.
func (s *Settings) SetString(setting string, to string) {
	cStrSetting := C.CString(setting)
	defer C.free(unsafe.Pointer(cStrSetting))
	cStrTo := C.CString(to)
	defer C.free(unsafe.Pointer(cStrTo))
	C._g_object_set_one(
		s.object(),
		(*C.gchar)(cStrSetting),
		unsafe.Pointer(&cStrTo))
}

// SetUint sets a single uint setting.
//
// Type checks are not made, if the setting isn't a uint, this method will
// panic.
func (s *Settings) SetUint(setting string, to uint) {
	cStr := C.CString(setting)
	defer C.free(unsafe.Pointer(cStr))
	cuint := C.guint(to)
	C._g_object_set_one(
		s.object(),
		(*C.gchar)(cStr),
		unsafe.Pointer(&cuint))
}

// GetBool gets a single boolean setting.
//
// Type checks are not made, if the setting isn't boolean, the behaviour is
// undefined.
func (s *Settings) GetBool(setting string) bool {
	cStr := C.CString(setting)
	defer C.free(unsafe.Pointer(cStr))
	cboolPtr := new(C.gboolean)
	C._g_object_get_one(
		s.object(),
		(*C.gchar)(cStr),
		unsafe.Pointer(cboolPtr))
	return gobool(*cboolPtr)
}

// GetString gets a single string setting.
//
// Type checks are not made, if the setting isn't a string, the behaviour is
// undefined.
func (s *Settings) GetString(setting string) string {
	cStr := C.CString(setting)
	defer C.free(unsafe.Pointer(cStr))
	cStrPtr := new(*C.gchar)
	defer C.free(unsafe.Pointer(*cStrPtr))
	C._g_object_get_one(
		s.object(),
		(*C.gchar)(cStr),
		unsafe.Pointer(cStrPtr))
	return C.GoString((*C.char)(*cStrPtr))
}

// GetUint gets a single uint setting.
//
// Type checks are not made, if the setting isn't a uint, the behaviour is
// undefined.
func (s *Settings) GetUint(setting string) uint {
	cStr := C.CString(setting)
	defer C.free(unsafe.Pointer(cStr))
	cuintPtr := new(C.guint)
	C._g_object_get_one(
		s.object(),
		(*C.gchar)(cStr),
		unsafe.Pointer(cuintPtr))
	return uint(*cuintPtr)
}

// GetSettingsType gets the type of a particular setting.
//
// If the passed setting doesn't exist, an error is returned instead.
func GetSettingsType(setting string) (reflect.Type, error) {
	if boolSetting[setting] {
		return typeBool, nil
	} else if stringSetting[setting] {
		return typeString, nil
	} else if uintSetting[setting] {
		return typeUint, nil
	}
	return nil, fmt.Errorf("Unknown setting: %v", setting)
}
