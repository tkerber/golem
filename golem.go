package main

import (
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/conformal/gotk3/gtk"
	"github.com/guelfey/go.dbus"
	"github.com/tkerber/golem/webkit"
)

const scrollbarHideCSS = `
html::-webkit-scrollbar{
	height:0px!important;
	width:0px!important;
}`

type golem struct {
	*cfg
	windows            []*window
	webViews           map[uint64]*webView
	userContentManager *webkit.UserContentManager
	closeChan          chan<- *window
	quit               chan bool
	sBus               *dbus.Conn
	wMutex             *sync.Mutex
}

func newGolem(sBus *dbus.Conn) (*golem, error) {
	ucm, err := webkit.NewUserContentManager()
	if err != nil {
		return nil, err
	}
	css, err := webkit.NewUserStyleSheet(
		scrollbarHideCSS,
		webkit.UserContentInjectTopFrame,
		webkit.UserStyleLevelUser,
		[]string{},
		[]string{})
	if err != nil {
		return nil, err
	}
	ucm.AddStyleSheet(css)

	closeChan := make(chan *window)
	quitChan := make(chan bool)

	g := &golem{
		defaultCfg,
		make([]*window, 0, 10),
		make(map[uint64]*webView, 500),
		ucm,
		closeChan,
		quitChan,
		sBus,
		new(sync.Mutex),
	}

	sigChan := make(chan *dbus.Signal, 100)
	sBus.Signal(sigChan)
	go g.watchSignals(sigChan)

	return g, nil
}

func (g *golem) watchSignals(c <-chan *dbus.Signal) {
	for sig := range c {
		switch sig.Name {
		case webExtenDBusInterface + ".VerticalPositionChanged":
			if !strings.HasPrefix(string(sig.Path), webExtenDBusPathPrefix) {
				continue
			}
			originId, err := strconv.ParseUint(
				string(sig.Path[len(webExtenDBusPathPrefix):len(sig.Path)]),
				0,
				64)
			if err == nil {
				g.updatePosition(
					originId,
					sig.Body[0].(int64),
					sig.Body[1].(int64))
			}
		}
	}
}

func (g *golem) closeWindow(w *window) {
	g.wMutex.Lock()
	defer g.wMutex.Unlock()
	// w points to the window which was closed. It will be removed
	// from golems window list, and in doing so will be deferenced.
	var i int
	found := false
	for i = range g.windows {
		if g.windows[i] == w {
			found = true
			break
		}
	}
	if !found {
		log.Printf("Close signal recieved for non-existant window. Dropping.")
	}

	// Delete item at index i from the slice.
	l := len(g.windows)
	copy(g.windows[i:l-1], g.windows[i+1:l])
	g.windows[l-1] = nil
	g.windows = g.windows[0 : l-1]

	// If no windows are left, golem exits
	if len(g.windows) == 0 {
		gtk.MainQuit()
		g.quit <- true
	}
}

func (g *golem) updatePosition(pageId uint64, top, height int64) {
	wv, ok := g.webViews[pageId]
	if !ok {
		log.Printf(
			"Attempted to update position of non-existent webpage %d!",
			pageId)
		return
	}
	wv.top = top
	wv.height = height
	for _, w := range g.windows {
		if wv.WebView == w.WebView {
			w.Top = top
			w.Height = height
			w.UpdateLocation()
		}
	}
}
