package main

// cfg contains the configuration options for golem not already contained
// elsewhere.
type cfg struct {
	searchEngines *searchEngines
	newTabPage    string
	scrollDelta   int
	profile       string
}

// The defaultCfg is used when golem is started, and typically overwritten
// with rc commands.
var defaultCfg = &cfg{
	defaultSearchEngines,
	"http://github.com/tkerber/golem",
	40,
	"default",
}
