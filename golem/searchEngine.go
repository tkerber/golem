package golem

import (
	"fmt"
	"net/url"
)

// searchEngines is a collection of all search engines registered, with name,
// and a default search engine.
type searchEngines struct {
	searchEngines       map[string]*searchEngine
	defaultSearchEngine *searchEngine
}

// searchURI converts a list of terms into a URI for the search.
func (s *searchEngines) searchURI(searchTerms []string) string {
	searchEngine := s.defaultSearchEngine
	if searchEngine == nil {
		// TODO put some sensible tutorial or something here.
		return "golem:no_search_engines"
	}
	e, ok := s.searchEngines[searchTerms[0]]
	if len(searchTerms) > 1 && ok {
		searchEngine = e
		searchTerms = searchTerms[1:]
	} else {
		searchTerms = searchTerms[0:]
	}
	return searchEngine.searchURI(searchTerms)
}

// A searchEngine is a struct describing - well, a search engine.
type searchEngine struct {
	fullName     string
	formatString string
}

// searchURI converts a list of terms into a URI for the search.
func (s *searchEngine) searchURI(searchTerms []string) string {
	// the reason the replace is done after the escape is that e.g.
	// + is also escaped. This is counter productive.
	searchTermStr := ""
	for i, searchTerm := range searchTerms {
		if i != 0 {
			searchTermStr += "+"
		}
		searchTermStr += url.QueryEscape(searchTerm)
	}
	return fmt.Sprintf(
		s.formatString,
		searchTermStr)
}
