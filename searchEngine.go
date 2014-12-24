package main

import (
	"fmt"
	"net/url"
)

type searchEngines struct {
	searchEngines       map[string]*searchEngine
	defaultSearchEngine *searchEngine
}

func (s *searchEngines) searchURI(searchTerms []string) string {
	searchEngine := s.defaultSearchEngine
	e, ok := s.searchEngines[searchTerms[0]]
	if len(searchTerms) > 1 && ok {
		searchEngine = e
		searchTerms = searchTerms[1:]
	} else {
		searchTerms = searchTerms[0:]
	}
	return searchEngine.searchURI(searchTerms)
}

type searchEngine struct {
	fullName      string
	formatString  string
	searchTermSep string
}

func (s *searchEngine) searchURI(searchTerms []string) string {
	// the reason the replace is done after the escape is that e.g.
	// + is also escaped. This is counter productive.
	searchTermStr := ""
	for i, searchTerm := range searchTerms {
		if i != 0 {
			searchTermStr += s.searchTermSep
		}
		searchTermStr += url.QueryEscape(searchTerm)
	}
	return fmt.Sprintf(
		s.formatString,
		searchTermStr)
}

var searchEnginesMap = map[string]*searchEngine{
	"d": &searchEngine{
		"DuckDuckGo",
		"https://duckduckgo.com/?q=%v",
		"+",
	},
	"g": &searchEngine{"Google",
		"https://google.com/search?q=%v",
		"+",
	},
	"w": &searchEngine{
		"Wikipedia",
		"http://en.wikipedia.org/wiki/Special:Serach?search=%v&go=Go",
		"+",
	},
	"wt": &searchEngine{
		"Wiktionary",
		"http://en.wiktionary.org/wiki/Special:Serach?search=%v&go=Go",
		"+",
	},
}

var defaultSearchEngines = &searchEngines{
	searchEnginesMap,
	searchEnginesMap["d"],
}
