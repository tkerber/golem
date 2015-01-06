package golem

// cfg contains the configuration options for golem not already contained
// elsewhere.
type cfg struct {
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
	_, err := Asset("srv/pdf.js/web/viewer.html")
	defaultCfg = &cfg{
		defaultSearchEngines,
		"http://github.com/tkerber/golem",
		40,
		"default",
		err == nil,
		500,
	}
}
