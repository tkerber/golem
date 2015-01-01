package ui

import "github.com/tkerber/golem/webkit"

// The WebView interface keeps methods the UI needs to call on web views.
type WebView interface {
	GetTop() int64
	GetHeight() int64
	GetWebView() *webkit.WebView
}
