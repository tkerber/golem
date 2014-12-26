package webkit

import "github.com/conformal/gotk3/glib"

// UriRequest wraps a WebKitURIRequest.
//
// It keeps track of a request for a specific URI.
type UriRequest struct {
	*glib.Object
}
