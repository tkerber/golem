package main

// cfg contains the configuration options for golem not already contained
// elsewhere.
type cfg struct {
	searchEngines *searchEngines
	newTabPage    string
	scrollDelta   int
	profile       string
}

// The defaultCfg is used when golem is started, and typically overwritten
// with rc commands.
var defaultCfg = &cfg{
	defaultSearchEngines,
	"http://github.com/tkerber/golem",
	40,
	"default",
}

// The defaultRc is a (temporary) collection of commands executed when golem
// is started.
//
// TODO move this out.
const defaultRc = `
" This is a comment.
set webkit:user-agent+=" golem/0 (Anvil of the Void)"

bind d  builtin:tabClose
bind r  builtin:reload
bind gg builtin:scrollToTop
bind G  builtin:scrollToBottom
bind j  builtin:scrollDown
bind k  builtin:scrollUp
bind h  builtin:tabPrev
bind l  builtin:tabNext
bind :  builtin:commandMode
bind i  builtin:insertMode
bind ,h builtin:goBack
bind ,l builtin:goForward
bind o  builtin:open
bind O  builtin:tabOpen
bind go builtin:editURI
bind gO builtin:tabEditURI
bind wo builtin:windowOpen
bind we builtin:windowEditURI
bind by "cmd:open youtube.com"

" Mediasource isn't supported enough for YouTube yet :(
set webkit:enable-mediasource=true
`
