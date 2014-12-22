package main

import (
	"log"
	"regexp"
)

var commands = map[string]func(*window, []string){
	"open": cmdOpen,
}

func logInvalidArgs(args []string) {
	log.Printf("Invalid arguments recieved for command %v.", args[0])
}

func cmdOpen(w *window, args []string) {
	if len(args) < 2 {
		logInvalidArgs(args)
		return
	}
	uri := args[1]
	if regexp.MustCompile(`\w+:.*`).MatchString(uri) && len(args) == 2 {
		// We have a (hopefully) sensable protocol already. keep it.
	} else if regexp.MustCompile(`\S+\.\S+`).MatchString(uri) && len(args) == 2 {
		// What we have looks like a uri, but is missing the protocol.
		// We add http to it.

		// TODO any good way to have this sensibly default to https where
		// possible?
		uri = "http://" + uri
	} else {
		searchEngine := w.parent.defaultSearchEngine
		s, ok := w.parent.searchEngines[args[1]]
		var searchTerms []string
		if len(args) > 2 && ok {
			searchEngine = s
			searchTerms = args[2:len(args)]
		} else {
			searchTerms = args[1:len(args)]
		}
		uri = searchEngine.searchURI(searchTerms)
	}
	//log.Printf(uri)
	w.WebView.LoadURI(uri)
}
