package cfg

import (
	"fmt"
	"net/url"
	"strings"
)

type Settings struct {
	DefaultSearchEngine *SearchEngine
	SearchEngines       map[string]*SearchEngine
	HomePage            string
	// TODO move mapping tree stuff here.
}

type SearchEngine struct {
	FullName      string
	FormatString  string
	ReplaceSpaces string
}

func (s *SearchEngine) SearchURI(searchTerm string) string {
	return fmt.Sprintf(
		s.FormatString,
		strings.Replace(
			url.QueryEscape(searchTerm),
			"%20",
			s.ReplaceSpaces,
			-1))
}

// These are temporary. Maybe.
var DefaultSearchEngines = map[string]*SearchEngine{
	"d":  &SearchEngine{"DuckDuckGo", "https://duckduckgo.com/?q=%v", "+"},
	"g":  &SearchEngine{"Google", "https://google.com/search?q=%v", "+"},
	"w":  &SearchEngine{"Wikipedia", "http://en.wikipedia.org/wiki/Special:Search?search=%v&go=Go", "+"},
	"wt": &SearchEngine{"Wiktionary", "http://en.wiktionary.org/wiki/Special:Search?search=%v&go=Go", "+"},
}

var DefaultSettings = &Settings{
	DefaultSearchEngines["d"],
	DefaultSearchEngines,
	"http://github.com/tkerber/golem",
}
