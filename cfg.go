package main

// cfg contains the configuration options for golem not already contained
// elsewhere.
type cfg struct {
	searchEngines *searchEngines
	homePage      string
	scrollDelta   int
}

// The defaultCfg is used when golem is started, and typically overwritten
// with rc commands.
var defaultCfg = &cfg{
	defaultSearchEngines,
	"http://github.com/tkerber/golem",
	40,
}

// The defaultRc is a (temporary) collection of commands executed when golem
// is started.
//
// TODO move this out.
const defaultRc = `
" This is a comment.
set webkit:user-agent+=" golem/0 (Anvil of the Void)"

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
