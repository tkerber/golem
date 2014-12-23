package webkit

// #cgo pkg-config: glib-2.0
// #include <glib.h>
import "C"
import (
	"errors"
)

var errNilPtr = errors.New("cgo returned unexpected nil pointer")

func cbool(b bool) C.gboolean {
	if b {
		return C.gboolean(1)
	}
	return C.gboolean(0)
}

func gobool(b C.gboolean) bool {
	return b != 0
}
