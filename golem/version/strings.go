// Package version contains golem's version constants.
package version

import (
	"fmt"
	"strings"
)

// A version string for golem.
//
// Of one of the following forms:
//
// tag
// tag[hash]
// [hash]
var Version string

// A version name for golem.
var Name string

func init() {
	if CommitTagIsExact {
		Version = CommitTag
	} else if CommitTag == "" {
		Version = fmt.Sprintf("[%s]", CommitHashShort)
	} else {
		Version = fmt.Sprintf("%s[%s]", CommitTag, CommitHashShort)
	}

	switch {
	case CommitTagIsExact && strings.HasPrefix(CommitTag, "v1."):
		Name = "Shale"
	case CommitTagIsExact && strings.HasPrefix(CommitTag, "v2."):
		Name = "Caridin"
	default:
		Name = "Anvil of the Void"
	}
}
