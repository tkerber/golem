package webkit

import (
	"errors"
)

var errNilPtr = errors.New("cgo returned unexpected nil pointer")
