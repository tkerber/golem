package webkit

import (
	"github.com/conformal/gotk3/glib"
)

// A BackForwardList is a list of elements to go back or forward in browser
// history to.
type BackForwardList struct {
	*glib.Object
}
