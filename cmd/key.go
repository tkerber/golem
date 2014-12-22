package cmd

// #cgo pkg-config: gdk-3.0
// #include <gdk/gdk.h>
// #include <stdlib.h>
/*
static guint
gdk_event_key_is_modifier(GdkEventKey *key) {
	return key->is_modifier;
}
*/
import "C"
import (
	"fmt"
	"strings"
	"unicode/utf8"
	"unsafe"

	"github.com/conformal/gotk3/gdk"
)

// nonPrintRunes are runes which shouldn't be printed by themselves,
// i.e. key names will be printed instead of them.
var nonPrintRunes = []rune{
	'\t',
	'\n',
	'\r',
	'\v',
	'\f',
	'\b',
}

// selectiveNonPrintRunes are runes which shouldn't be printed in some contexts
// (e.g. describing a key sequence), but should be in others (e.g. in a
// command-line input.
var selectiveNonPrintRunes = []rune{
	' ',
	'<',
}

const (
	ShiftMask   = C.GDK_SHIFT_MASK
	LockMask    = C.GDK_LOCK_MASK
	ControlMask = C.GDK_CONTROL_MASK
	Mod1Mask    = C.GDK_MOD1_MASK
	Mod2Mask    = C.GDK_MOD2_MASK
	Mod3Mask    = C.GDK_MOD3_MASK
	Mod4Mask    = C.GDK_MOD4_MASK
	Mod5Mask    = C.GDK_MOD5_MASK
	Button1Mark = C.GDK_BUTTON1_MASK
	Button2Mark = C.GDK_BUTTON2_MASK
	Button3Mark = C.GDK_BUTTON3_MASK
	Button4Mark = C.GDK_BUTTON4_MASK
	Button5Mark = C.GDK_BUTTON5_MASK
	SuperMask   = C.GDK_SUPER_MASK
	HyperMask   = C.GDK_HYPER_MASK
	MetaMask    = C.GDK_META_MASK
	ReleaseMask = C.GDK_RELEASE_MASK

	modifierCmpMask = ControlMask | Mod1Mask
)

const (
	KeyVoid      = C.GDK_KEY_VoidSymbol
	KeyEscape    = C.GDK_KEY_Escape
	KeyReturn    = C.GDK_KEY_Return
	KeyBackSpace = C.GDK_KEY_BackSpace
)

// isNonPrintRune checks in a rune is a member of nonPrintRunes.
//
// If selective is true, it also returns true if a rune is a member of
// selectiveNonPrintRunes.
func isNonPrintRune(r rune, selective bool) bool {
	for _, r2 := range nonPrintRunes {
		if r == r2 {
			return true
		}
	}
	if selective {
		for _, r2 := range selectiveNonPrintRunes {
			if r == r2 {
				return true
			}
		}
	}
	return false
}

type keyParseError string

func (e keyParseError) Error() string {
	return fmt.Sprintf("Failed to parse key for value: %v", e)
}

type Key struct {
	Keyval     uint
	Modifiers  uint
	IsModifier bool
}

func (k Key) Match(k2 Key) bool {
	return k.Normalize() == k2.Normalize()
}

// Normalize normalizes a key by masking out all modifiers except for
// control and alt.
func (k Key) Normalize() Key {
	return Key{k.Keyval, k.Modifiers & modifierCmpMask, k.IsModifier}
}

func NewKeyFromEventKey(ek gdk.EventKey) Key {
	cek := (*C.GdkEventKey)(unsafe.Pointer(ek.Native()))
	return Key{
		uint(cek.keyval),
		uint(cek.state),
		C.gdk_event_key_is_modifier(cek) != 0,
	}
}

// NewKeyFromString creates a new key object from a string.
//
// Note that Key objects created for modifier keys will be incorrectly
// flagged as not being modifiers. This functionality is at the time of
// writing not required.
func NewKeyFromString(strOrig string) (Key, error) {
	str := strOrig
	var mod uint = 0
	for len(str) >= 2 {
		switch str[0:2] {
		case "C-":
			mod |= ControlMask
		case "A-":
			mod |= Mod1Mask
		default:
			return Key{0, 0, false}, keyParseError(strOrig)
		}
		str = str[2:len(str)]
	}
	var keyval uint
	if utf8.RuneCountInString(str) == 1 {
		r, _ := utf8.DecodeRuneInString(str)
		keyval = uint(C.gdk_unicode_to_keyval(C.guint32(r)))
	} else {
		cStr := (*C.gchar)(C.CString(str))
		defer C.free(unsafe.Pointer(cStr))
		keyval = uint(C.gdk_keyval_from_name(cStr))
	}
	if keyval == KeyVoid {
		return Key{0, 0, false}, keyParseError(strOrig)
	}
	return Key{keyval, mod, false}, nil
}

func (k Key) StringSelective(selective bool) string {
	// Produces string like "a", "C-a", "C-A-a", "Escape", "C-Escape"
	str := ""

	if (k.Modifiers & ControlMask) != 0 {
		str += "C-"
	}
	if (k.Modifiers & Mod1Mask) != 0 {
		str += "A-"
	}

	r := rune(C.gdk_keyval_to_unicode(C.guint(k.Keyval)))
	if r != 0 && !isNonPrintRune(r, selective) {
		return str + string(r)
	}
	cStr := C.gdk_keyval_name(C.guint(k.Keyval))
	return str + C.GoString((*C.char)(cStr))
}

func (k Key) String() string {
	return k.StringSelective(true)
}

func KeysStringSelective(keys []Key, selective bool) string {
	str := ""
	for _, key := range keys {
		keyStr := key.StringSelective(selective)
		if len(keyStr) == 1 {
			str += keyStr
		} else {
			str += "<" + keyStr + ">"
		}
	}
	return str
}

func KeysString(keys []Key) string {
	return KeysStringSelective(keys, true)
}

func ParseKeys(str string) ([]Key, error) {
	keys := make([]Key, 0)
	for len(str) > 0 {
		// For now, < *cannot* be a key by itself unless no > (after it) is
		// contained. Use <less> instead.
		if str[0] == '<' {
			end := strings.IndexRune(str, '>')
			// If '>' isn't found, we fall through to the handling on the
			// typical, single-character key.
			if end != -1 {
				key, err := NewKeyFromString(str[1:end])
				if err != nil {
					return nil, err
				}
				keys = append(keys, key)
				str = str[end+1 : len(str)]
				continue
			}
		}
		// Note no else here. This is due to the continue and the comment
		// above.
		r, rLen := utf8.DecodeRuneInString(str)
		key, err := NewKeyFromString(string(r))
		// Really no errors should occur. But hey.
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
		str = str[rLen:len(str)]
	}
	return keys, nil
}
