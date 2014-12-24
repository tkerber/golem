package webkit

// #cgo pkg-config: glib-2.0
// #include <glib.h>
import "C"
import (
	"errors"
)

// errNilPtr is an error representing an unexpected nil pointer.
var errNilPtr = errors.New("cgo returned unexpected nil pointer")

// cbool converts a go bool into a gboolean.
func cbool(b bool) C.gboolean {
	if b {
		return C.gboolean(1)
	}
	return C.gboolean(0)
}

// gobool converts a gboolean into a go bool.
func gobool(b C.gboolean) bool {
	return b != 0
}
