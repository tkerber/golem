package main

import (
	"fmt"
	"net/url"
)

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

var defaultSearchEngines = map[string]*searchEngine{
	"d": &searchEngine{"DuckDuckGo", "https://duckduckgo.com/?q=%v", "+"},
	"g": &searchEngine{"Google", "https://google.com/search?q=%v", "+"},
	"w": &searchEngine{
		"Wikipedia",
		"http://en.wikipedia.org/wiki/Special:Serach?search=%v&go=Go",
		"+"},
	"wt": &searchEngine{
		"Wiktionary",
		"http://en.wiktionary.org/wiki/Special:Serach?search=%v&go=Go",
		"+"},
}
