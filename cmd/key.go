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
	"strconv"
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

// A Modifiers is a mask of GDK modifier keys.
//
// They are represented as a standard bitmask.
type Modifiers uint

// The constants in this block map directly to GDK modifier masks; each
// represents a modifier key or button, which may be pressed.
//
// They may be logically ORed to combine them, and logically ANDed to check
// whether a given Key has these modifiers pressed.
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

	// The modifiers which are considered for comparison operations, all other
	// modifiers are ignored.
	modifierNormalMask = ControlMask | Mod1Mask
)

// The constants in this block map directly to GDK keyvals, and are used to
// compare with Key.Keyval to check which key was pressed.
const (
	KeyVoid      = C.GDK_KEY_VoidSymbol
	KeyEscape    = C.GDK_KEY_Escape
	KeyLeft      = C.GDK_KEY_Left
	KeyKPLeft    = C.GDK_KEY_KP_Left
	KeyRight     = C.GDK_KEY_Right
	KeyKPRight   = C.GDK_KEY_KP_Right
	KeyReturn    = C.GDK_KEY_Return
	KeyKPEnter   = C.GDK_KEY_KP_Enter
	KeyBackSpace = C.GDK_KEY_BackSpace
	KeyDelete    = C.GDK_KEY_Delete
	KeyKPDelete  = C.GDK_KEY_KP_Delete
	KeyHome      = C.GDK_KEY_Home
	KeyKPHome    = C.GDK_KEY_KP_Home
	KeyEnd       = C.GDK_KEY_End
	KeyKPEnd     = C.GDK_KEY_KP_End
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

// keyParseError is an error parsing a key.
//
// Its string value is the string which failed to parse.
type keyParseError string

// Error returns the error message associated with this parse error.
func (e keyParseError) Error() string {
	return fmt.Sprintf("Failed to parse key for value: %v", e)
}

// A Key is the representation of a single key, real or virtual.
type Key interface {
	KeyType() KeyType
	Equals(Key, Modifiers) bool
	String() string
	StringSelective(bool) string
}

// A KeyType is the type of a specific key.
type KeyType uint

const (
	// KeyTypeNormal is a normal, physical key.
	KeyTypeNormal KeyType = iota
	// KeyTypeModifier is a modifier key.
	KeyTypeModifier
	// KeyTypeVirtual is a non-existant, virtual key.
	KeyTypeVirtual
)

// A RealKey is a keyval combined with the pressed modifiers.
//
// It can be derived from a key event, or simply be an abstract representation
// or a key.
type RealKey struct {
	Keyval     uint
	Modifiers  Modifiers
	IsModifier bool
}

// IsNum checks if a real key is a number key.
func (k RealKey) IsNum() bool {
	str := k.String()
	return len(str) == 1 && str[0] >= '0' && str[0] <= '9'
}

// NumVal retrieves the numeric value of a single key.
func (k RealKey) NumVal() (int, error) {
	i, err := strconv.ParseInt(k.String(), 10, 0)
	return int(i), err
}

// KeyType gets the type of the key.
func (k RealKey) KeyType() KeyType {
	if k.IsModifier {
		return KeyTypeModifier
	}
	return KeyTypeNormal
}

// Equals compares to another key for equality, respecting a particular
// modifier mask. (i.e. only these modifiers will be compared)
func (k RealKey) Equals(k2 Key, mods Modifiers) bool {
	k2r, ok := k2.(RealKey)
	if !ok {
		return false
	}
	return k.Keyval == k2r.Keyval && (k.Modifiers&mods) == (k2r.Modifiers&mods)
}

// StringSelective produces a string value associated with a key, optionally
// forcing selected Keys into their long form.
//
// In particular, if selective is true, '<' is written as <less> and ' ' is
// written as <space>
//
// Keys can be in a short form ('a', '!', '/') or long form ('Escape', 'Tab',
// 'Enter').
//
// The short form will be used for most Keys with an associated character,
// with the exception of whitespace (except the literal space, which depends
// on the selective parameter), and (again with the selective parameter) a '<'.
//
// The long form will be used in all other cases.
//
// If a Key has the control modifier pressed, 'C-' is prepended. Likewise, if
// alt is pressed, 'A-' is prepended. 'C-A-' is prepended if both are pressed.
func (k RealKey) StringSelective(selective bool) string {
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

// String produces a string value associated with a key, forcing selected keys
// into their long form.
//
// See Key.StringSelective
func (k RealKey) String() string {
	return k.StringSelective(true)
}

// Normalize keeps only Control and Alt modfiers of the key.
func (k RealKey) Normalize() RealKey {
	return RealKey{k.Keyval, k.Modifiers & modifierNormalMask, k.IsModifier}
}

// NewKeyFromEventKey converts a gdk key event into a Key.
func NewKeyFromEventKey(ek gdk.EventKey) RealKey {
	cek := (*C.GdkEventKey)(unsafe.Pointer(ek.Native()))
	return RealKey{
		uint(cek.keyval),
		Modifiers(cek.state),
		C.gdk_event_key_is_modifier(cek) != 0,
	}
}

// A VirtualKey is the abstract notion of a named key.
type VirtualKey string

// Equals compares this virtual key with another.
func (k VirtualKey) Equals(k2 Key, mods Modifiers) bool {
	return k == k2
}

// KeyType retrieves the type of this key.
//
// Will always return KeyTypeVirtual.
func (k VirtualKey) KeyType() KeyType {
	return KeyTypeVirtual
}

// String retrieves the string value of this key.
func (k VirtualKey) String() string {
	return string(k)
}

// StringSelective retrieves the string value of this key, selectively printing
// certain keys in their long form.
//
// Equivalent to String for VirtualKeys.
func (k VirtualKey) StringSelective(selective bool) string {
	return string(k)
}

// NewKeyFromRune creates a new key object from a rune.
func NewKeyFromRune(r rune) Key {
	keyval := uint(C.gdk_unicode_to_keyval(C.guint32(r)))
	return RealKey{keyval, 0, false}
}

// NewKeyFromString creates a new key object from a string.
//
// Note that Key objects created for modifier keys will be incorrectly
// flagged as not being modifiers. This functionality is at the time of
// writing not required.
//
// If the string Starts with C-, A-, C-A- or A-C-, it will be interpreted
// as the modifiers control, alt, both or both being pressed respectively.
//
// Beyond such a prefix, a key is either parsed as whichever key is associated
// with the single unicode rune remaining, (e.g. a or ! or £), or whichever
// key has the name of the string remaining (e.g. Escape, Enter, space)
//
// Note the importance of capitalization.
//
// If a real key cannot be derived in this way, a virtual one is used instead.
func NewKeyFromString(strOrig string) Key {
	str := strOrig
	var mod Modifiers
loop:
	for len(str) >= 2 {
		switch str[0:2] {
		case "C-":
			mod |= ControlMask
		case "A-":
			mod |= Mod1Mask
		default:
			// We've probably got a key name.
			break loop
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
		// It's not a real key. So lets make it virtual.
		return VirtualKey(strOrig)
	}
	return RealKey{keyval, mod, false}
}

// KeysStringSelective produces a string representation of a slice of keys,
// selectively forcing some into their long form.
//
// Each Key will be handled as in Key.StringSelective.
//
// Keys producing a string value longer than one character will be placed in
// angled braces - e.g. <Escape> or <C-a>
func KeysStringSelective(keys []Key, selective bool) string {
	str := ""
	for _, key := range keys {
		keyStr := key.StringSelective(selective)
		if utf8.RuneCountInString(keyStr) == 1 {
			str += keyStr
		} else {
			str += "<" + keyStr + ">"
		}
	}
	return str
}

// KeysString produces a string value associated with a slice of keys, forcing
// selected keys into their long form.
//
// See KeysStringSelective
func KeysString(keys []Key) string {
	return KeysStringSelective(keys, true)
}

// ParseKeys parses a string into the slice of Keys it represents. Each
// individual key is parsed as in NewKeyFromString.
//
// Each individual key is either in angled braces (e.g. <Escape>), or a single
// unicode rune (e.g. a, $, £). To avoid ambiguity, a left angle brace '<' is
// only parsed as a single key if no right angle braces follow it. If it is
// necessary to used it in such a situation, write out <left> instead.
func ParseKeys(str string) []Key {
	var keys []Key
	for len(str) > 0 {
		// For now, < *cannot* be a key by itself unless no > (after it) is
		// contained. Use <less> instead.
		if str[0] == '<' {
			end := strings.IndexRune(str, '>')
			// If '>' isn't found, we fall through to the handling on the
			// typical, single-character key.
			if end != -1 {
				key := NewKeyFromString(str[1:end])
				keys = append(keys, key)
				str = str[end+1 : len(str)]
				continue
			}
		}
		// Note no else here. This is due to the continue and the comment
		// above.
		r, rLen := utf8.DecodeRuneInString(str)
		key := NewKeyFromString(string(r))
		keys = append(keys, key)
		str = str[rLen:len(str)]
	}
	return keys
}
