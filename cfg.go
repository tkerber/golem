package main

type cfg struct {
	searchEngines *searchEngines
	homePage      string
	scrollDelta   int
}

var defaultCfg = &cfg{
	defaultSearchEngines,
	"http://github.com/tkerber/golem",
	40,
}

const defaultRc = `
" This is a comment.
set webkit:user-agent+=" golem/0 (voidAnvil)"

bind r  builtin:reload
bind gh builtin:goHome
bind gg builtin:scrollToTop
bind G  builtin:scrollToBottom
bind j  builtin:scrollDown
bind k  builtin:scrollUp
bind h  builtin:scrollLeft
bind l  builtin:scrollRight
bind :  builtin:commandMode
bind i  builtin:insertMode
bind ,h builtin:goBack
bind ,l builtin:goForward
bind o  builtin:open
bind go builtin:editURI
bind by "cmd:open youtube.com"

" Mediasource isn't supported enough for YouTube yet :(
set webkit:enable-mediasource=true
`
