package main

import (
	"log"

	"github.com/conformal/gotk3/gtk"
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
	userContentManager *webkit.UserContentManager
	closeChan          chan<- *window
	openChan           chan<- *window
	quit               chan bool
}

func newGolem() (*golem, error) {
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
	openChan := make(chan *window)
	quitChan := make(chan bool)

	g := &golem{
		defaultCfg,
		make([]*window, 0),
		ucm,
		closeChan,
		openChan,
		quitChan,
	}

	// This goroutine manages any writes to the golem struct itself,
	// protecting against concurrent access.
	go func() {
		for {
			select {
			case w := <-closeChan:
				g.closeWindow(w)
			case w := <-openChan:
				g.windows = append(g.windows, w)
			}
		}
	}()

	return g, nil
}

func (g *golem) closeWindow(w *window) {
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
