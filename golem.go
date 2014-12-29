package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/conformal/gotk3/gtk"
	"github.com/guelfey/go.dbus"
	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/webkit"
)

// scrollbarHideCSS is the CSS to hide the scroll bars for webkit.
const scrollbarHideCSS = `
html::-webkit-scrollbar{
	height:0px!important;
	width:0px!important;
}`

// golem is golem's main instance.
type golem struct {
	*cfg
	windows            []*window
	webViews           map[uint64]*webView
	userContentManager *webkit.UserContentManager
	closeChan          chan<- *window
	quit               chan bool
	sBus               *dbus.Conn
	wMutex             *sync.Mutex
	rawBindings        []cmd.RawBinding
	defaultSettings    *webkit.Settings
	files              *files
	extenDir           string
}

// newGolem creates a new instance of golem.
func newGolem(sBus *dbus.Conn, profile string) (*golem, error) {
	ucm, err := webkit.NewUserContentManager()
	if err != nil {
		return nil, err
	}
	css, err := webkit.NewUserStyleSheet(
		scrollbarHideCSS,
		webkit.UserContentInjectTopFrame,
		webkit.UserStyleLevelUser,
		nil,
		nil)
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
		make([]cmd.RawBinding, 0, 100),
		webkit.NewSettings(),
		nil,
		"",
	}
	g.profile = profile

	g.files, err = g.newFiles()
	if err != nil {
		return nil, err
	}

	g.webkitInit()

	sigChan := make(chan *dbus.Signal, 100)
	sBus.Signal(sigChan)
	go g.watchSignals(sigChan)

	rc, err := g.files.readRC()
	if err != nil {
		return nil, err
	}
	for _, rcLine := range strings.Split(rc, "\n") {
		runCmd(nil, g, rcLine)
	}

	return g, nil
}

// bind creates a new key binding.
func (g *golem) bind(from string, to string) {
	// We check if the key has been bound before. If so, we replace the
	// binding.
	index := -1
	for i, b := range g.rawBindings {
		if from == b.From {
			index = i
			break
		}
	}

	g.wMutex.Lock()
	if index != -1 {
		g.rawBindings[index] = cmd.RawBinding{from, to}
	} else {
		g.rawBindings = append(g.rawBindings, cmd.RawBinding{from, to})
	}
	g.wMutex.Unlock()

	for _, w := range g.windows {
		w.rebuildBindings()
	}
}

// watchSignals watches all DBus signals coming in through a channel, and
// handles them appropriately.
func (g *golem) watchSignals(c <-chan *dbus.Signal) {
	for sig := range c {
		if !strings.HasPrefix(
			string(sig.Path),
			fmt.Sprintf(webExtenDBusPathPrefix, g.profile)) {

			continue
		}
		originId, err := strconv.ParseUint(
			string(sig.Path[len(
				fmt.Sprintf(webExtenDBusPathPrefix, g.profile)):len(sig.Path)]),
			0,
			64)
		if err != nil {
			continue
		}
		wv, ok := g.webViews[originId]
		if !ok {
			continue
		}
		switch sig.Name {
		case webExtenDBusInterface + ".VerticalPositionChanged":
			// Update for bookkeeping when tabs are switched
			wv.top = sig.Body[0].(int64)
			wv.height = sig.Body[1].(int64)
			// Update any windows with this webview displayed.
			for _, w := range g.windows {
				if wv.WebView == w.WebView {
					w.Top = wv.top
					w.Height = wv.height
					w.UpdateLocation()
				}
			}
		case webExtenDBusInterface + ".InputFocusChanged":
			focused := sig.Body[0].(bool)
			// If it's newly focused, set any windows with this webview
			// displayed to insert mode.
			//
			// Otherwise, if the window is currently in insert mode and it's
			// newly unfocused, set this webview to normal mode.
			for _, w := range g.windows {
				if wv.WebView == w.WebView {
					if focused {
						w.setState(cmd.NewInsertMode(w.State))
					} else if _, ok := w.State.(*cmd.InsertMode); ok {
						w.setState(
							cmd.NewNormalMode(w.State))
					}
				}
			}
		}
	}
}

// closeWindow updates bookkeeping after a window was closed.
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
