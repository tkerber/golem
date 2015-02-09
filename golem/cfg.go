package golem

// cfg contains the configuration options for golem not already contained
// elsewhere.
type cfg struct {
	// A map from sanitized keystring (i.e. parsed and stringified again) to
	// a struct of uri and name.
	quickmarks   map[string]uriEntry
	hasQuickmark map[string]bool

	bookmarks  []uriEntry
	isBookmark map[string]bool

	searchEngines *searchEngines
	newTabPage    string
	scrollDelta   int
	profile       string
	pdfjsEnabled  bool
	maxHistLen    int
}

// The defaultCfg is used when golem is started, and typically overwritten
// with rc commands.
var defaultCfg *cfg

// init initializes the default config.
func init() {
	_, err := Asset("srv/pdf.js/enabled")
	defaultCfg = &cfg{
		make(map[string]uriEntry, 20),
		make(map[string]bool, 20),
		make([]uriEntry, 0, 100),
		make(map[string]bool, 100),
		&searchEngines{make(map[string]*searchEngine, 10), nil},
		"http://github.com/tkerber/golem",
		40,
		"default",
		err == nil,
		500,
	}
}
