package main

type cfg struct {
	defaultSearchEngine *searchEngine
	searchEngines       map[string]*searchEngine
	homePage            string
}

var defaultCfg = &cfg{
	defaultSearchEngines["d"],
	defaultSearchEngines,
	"http://github.com/tkerber/golem",
}
