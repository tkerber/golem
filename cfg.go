package main

type cfg struct {
	defaultSearchEngine *searchEngine
	searchEngines       map[string]*searchEngine
	homePage            string
	scrollDelta         int
}

var defaultCfg = &cfg{
	defaultSearchEngines["d"],
	defaultSearchEngines,
	"http://github.com/tkerber/golem",
	40,
}

const defaultRc = `
" This is a comment.
bind r  ::builtin:reload
bind gh ::builtin:goHome
bind gg ::builtin:scrollToTop
bind G  ::builtin:scrollToBottom
bind j  ::builtin:scrollDown
bind k  ::builtin:scrollUp
bind h  ::builtin:scrollLeft
bind l  ::builtin:scrollRight
bind :  ::builtin:commandMode
bind i  ::builtin:insertMode
bind ,h ::builtin:goBack
bind ,l ::builtin:goForward
bind o  ::builtin:open
bind go ::builtin:editURI
`
