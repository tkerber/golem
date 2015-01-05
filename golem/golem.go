package golem

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

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

// Golem is golem's main instance.
type Golem struct {
	*cfg
	windows            []*Window
	webViews           map[uint64]*webView
	userContentManager *webkit.UserContentManager
	closeChan          chan<- *Window
	Quit               chan bool
	sBus               *dbus.Conn
	wMutex             *sync.Mutex
	rawBindings        []cmd.RawBinding
	// A map from sanitized keystring (i.e. parsed and stringified again) to
	// uris.
	quickmarks   map[string]string
	hasQuickmark map[string]bool

	DefaultSettings *webkit.Settings
	files           *files
	extenDir        string

	webViewCache     []*webView
	webViewCacheFree chan bool
}

// New creates a new instance of golem.
func New(sBus *dbus.Conn, profile string) (*Golem, error) {
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

	closeChan := make(chan *Window)
	quitChan := make(chan bool)

	g := &Golem{
		defaultCfg,
		make([]*Window, 0, 10),
		make(map[uint64]*webView, 500),
		ucm,
		closeChan,
		quitChan,
		sBus,
		new(sync.Mutex),
		make([]cmd.RawBinding, 0, 100),
		make(map[string]string, 20),
		make(map[string]bool, 20),
		webkit.NewSettings(),
		nil,
		"",
		make([]*webView, 0),
		make(chan bool, 1),
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

	for _, rcfile := range g.files.rcFiles() {
		err := g.useRcFile(rcfile)
		if err != nil {
			return nil, err
		}
	}

	return g, nil
}

// cutWebViews moves the supplied web views to an internal buffer, and keeps
// them there for at most 1 minute.
func (g *Golem) cutWebViews(wvs []*webView) {
	g.wMutex.Lock()
	defer g.wMutex.Unlock()
	g.webViewCacheFree <- true
	g.webViewCacheFree = make(chan bool, 1)
	g.webViewCache = wvs
	cp := make([]*webView, len(g.webViewCache))
	copy(cp, g.webViewCache)
	go func() {
		select {
		case <-time.After(time.Minute):
			g.wMutex.Lock()
			g.webViewCache = make([]*webView, 0)
			g.wMutex.Unlock()
		case free := <-g.webViewCacheFree:
			if !free {
				return
			}
		}
		for _, wv := range cp {
			wv.close()
		}
	}()
}

// pasteWebViews retrieves the contents of the web view cache and resets it
// safely.
func (g *Golem) pasteWebViews() (wvs []*webView) {
	g.wMutex.Lock()
	defer g.wMutex.Unlock()
	g.webViewCacheFree <- false
	g.webViewCacheFree = make(chan bool, 1)
	wvs = g.webViewCache
	g.webViewCache = make([]*webView, 0)
	return wvs
}

// useRcFile reads and executes an rc file.
func (g *Golem) useRcFile(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		runCmd(nil, g, scanner.Text())
	}
	return nil
}

// bind creates a new key binding.
func (g *Golem) bind(from string, to string) {
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

// quickmark adds a quickmark to golem.
func (g *Golem) quickmark(from string, uri string) {
	g.wMutex.Lock()
	defer g.wMutex.Unlock()
	g.quickmarks[from] = uri
	g.hasQuickmark[uri] = true

	for _, w := range g.windows {
		w.rebuildQuickmarks()
	}
}

// watchSignals watches all DBus signals coming in through a channel, and
// handles them appropriately.
func (g *Golem) watchSignals(c <-chan *dbus.Signal) {
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
				if wv == w.getWebView() {
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
				if wv == w.getWebView() {
					if focused {
						w.setState(cmd.NewInsertMode(w.State, cmd.SubstateDefault))
					} else if _, ok := w.State.(*cmd.InsertMode); ok {
						w.setState(
							cmd.NewNormalMode(w.State))
					}
				}
			}
		}
	}
}

// Close closes golem.
func (g *Golem) Close() {
	for _, w := range g.windows {
		w.Close()
	}
}

// closeWindow updates bookkeeping after a window was closed.
func (g *Golem) closeWindow(w *Window) {
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
		(*Window)(nil).logError("Close signal recieved for non-existant " +
			"window. Dropping.")
	}

	// Delete item at index i from the slice.
	l := len(g.windows)
	copy(g.windows[i:l-1], g.windows[i+1:l])
	g.windows[l-1] = nil
	g.windows = g.windows[0 : l-1]

	// If no windows are left, golem exits
	if len(g.windows) == 0 {
		gtk.MainQuit()
		g.Quit <- true
	}
}

// addDownload adds a new download to the tracked downloads.
func (g *Golem) addDownload(d *webkit.Download) {
}
