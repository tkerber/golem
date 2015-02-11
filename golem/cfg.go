package golem

// globalCfg contains the configuration for the entire golem session.
type globalCfg struct {
	*windowCfg

	// A map from sanitized keystring (i.e. parsed and stringified again) to
	// a struct of uri and name.
	quickmarks   map[string]uriEntry
	hasQuickmark map[string]bool

	bookmarks  []uriEntry
	isBookmark map[string]bool

	searchEngines *searchEngines

	profile      string
	pdfjsEnabled bool
	maxHistLen   int
}

// windowCfg contains the configuration for a single window.
type windowCfg struct {
	*tabCfg

	newTabPage string
}

// clone clones the config.
func (c *windowCfg) clone() *windowCfg {
	return &windowCfg{
		c.tabCfg.clone(),
		c.newTabPage,
	}
}

// tabCfg contains the configuration for a single tab.
type tabCfg struct {
	scrollDelta int
}

// clone clones the config.
func (c *tabCfg) clone() *tabCfg {
	return &tabCfg{
		c.scrollDelta,
	}
}

// The defaultCfg is used when golem is started, and typically overwritten
// with rc commands.
var defaultCfg *globalCfg

// init initializes the default config.
func init() {
	_, err := Asset("srv/pdf.js/enabled")
	defaultCfg = &globalCfg{
		&windowCfg{
			&tabCfg{
				40,
			},
			"http://github.com/tkerber/golem",
		},
		make(map[string]uriEntry, 20),
		make(map[string]bool, 20),
		make([]uriEntry, 0, 100),
		make(map[string]bool, 100),
		&searchEngines{make(map[string]*searchEngine, 10), nil},
		"default",
		err == nil,
		500,
	}
}
