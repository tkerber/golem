package webkit

import "github.com/conformal/gotk3/glib"

// URIRequest wraps a WebKitURIRequest.
//
// It keeps track of a request for a specific URI.
type URIRequest struct {
	*glib.Object
}
