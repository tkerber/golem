package main

import (
	"log"
	"regexp"
)

var commands = map[string]func(*window, *golem, []string){
	"open": cmdOpen,
	"bind": cmdBind,
}

func logInvalidArgs(args []string) {
	log.Printf("Invalid arguments recieved for command %v.", args[0])
}

func logNonGlobalCommand() {
	log.Printf("Non global command executed in a global contex.")
}

func cmdOpen(w *window, g *golem, args []string) {
	if w == nil {
		logNonGlobalCommand()
	}

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
		searchEngine := g.defaultSearchEngine
		s, ok := g.searchEngines[args[1]]
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

// cmdBind adds a binding, globally to golem.
func cmdBind(w *window, g *golem, args []string) {
	if len(args) != 3 {
		logInvalidArgs(args)
		return
	}
	g.bind(args[1], args[2])
}
