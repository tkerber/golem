package main

import (
	"fmt"
	"net/url"
	"strings"
)

type searchEngine struct {
	fullName      string
	formatString  string
	replaceSpaces string
}

func (s *searchEngine) searchURI(searchTerm string) string {
	// the reason the replace is done after the escape is that e.g.
	// + is also escaped. This is counter productive.
	return fmt.Sprintf(
		s.formatString,
		strings.Replace(
			url.QueryEscape(searchTerm),
			"%20",
			s.replaceSpaces,
			-1))
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
