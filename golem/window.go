package golem

// #cgo pkg-config: gdk-3.0
// #include <gdk/gdk.h>
import "C"
import (
	"fmt"
	"log"
	"os"
	"regexp"
	"sync"
	"time"
	"unsafe"

	"github.com/conformal/gotk3/gdk"
	"github.com/conformal/gotk3/glib"
	"github.com/conformal/gotk3/gtk"
	"github.com/mattn/go-shellwords"
	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/golem/states"
	"github.com/tkerber/golem/golem/ui"
	"github.com/tkerber/golem/webkit"
)

// Errlog is a log pointed to stderr.
var Errlog = log.New(os.Stderr, "(E) ", log.LstdFlags)

// PrintKeys specifies whether each keypress should be printed.
var PrintKeys bool = false

// PrintCommands specifies whether all commands should be printed.
var PrintCommands bool = false

// blankLineRegex matches a blank or comment line for commands.
var blankLineRegex = regexp.MustCompile(`^\s*(".*|)$`)

// signalHandle is struct containing both a signal handle and the glib Object
// it applies to.
type signalHandle struct {
	obj    *glib.Object
	handle glib.SignalHandle
}

func newSignalHandle(obj *glib.Object, handle glib.SignalHandle, err error) *signalHandle {
	// failed connects won't cause any errors, but *will* be logged.
	if obj != nil && err == nil {
		return &signalHandle{obj, handle}
	}
	(*Window)(nil).logError("Broken signal handle dropped...")
	return nil
}

// disconnect disconnects the signal handle.
func (h *signalHandle) disconnect() {
	if h != nil {
		h.obj.HandlerDisconnect(h.handle)
	}
}

// A Window is one of golem's window.
type Window struct {
	*ui.Window
	cmd.State
	webViews            []*webView
	currentWebView      int
	parent              *Golem
	builtins            cmd.Builtins
	bindings            map[cmd.Substate]*cmd.BindingTree
	activeSignalHandles []*signalHandle
	windowSignalHandles []*signalHandle
	timeoutChan         chan bool
	wMutex              *sync.Mutex
}

// keyTimeout is the timeout between two key presses where no key press is
// handled.
//
// This is due to webkit re-raising key events, leading to them being recieved
// twice in close succession.
const keyTimeout = time.Millisecond * 10

// setState sets the windows state.
func (w *Window) setState(state cmd.State) {
	w.State = state
	w.UpdateState(w.State)
}

// newWindow creates a new window, using particular webkit settings as a
// template.
//
// A new web view is initialized and sent to a specified uri. If the URI is
// empty, the new tab page is used instead.
func (g *Golem) NewWindow(uri string) (*Window, error) {
	win := &Window{
		nil,
		nil,
		make([]*webView, 1, 50),
		0,
		g,
		nil,
		make(map[cmd.Substate]*cmd.BindingTree),
		make([]*signalHandle, 0),
		make([]*signalHandle, 0, 5),
		make(chan bool, 1),
		new(sync.Mutex),
	}

	var err error

	win.webViews[0], err = win.newWebView(g.DefaultSettings)
	if err != nil {
		return nil, err
	}

	win.Window, err = ui.NewWindow(win.webViews[0])
	if err != nil {
		return nil, err
	}

	tabUI, err := win.Window.AppendTab()
	if err != nil {
		return nil, err
	}
	win.webViews[0].setTabUI(tabUI)
	win.Window.FocusTab(0)

	win.builtins = builtinsFor(win)

	win.setState(cmd.NewState(win.bindings, win.setState))

	win.rebuildBindings()
	win.rebuildQuickmarks()

	g.wMutex.Lock()
	g.windows = append(g.windows, win)
	g.wMutex.Unlock()

	win.reconnectWebViewSignals()

	if uri == "" {
		win.webViews[0].LoadURI(g.newTabPage)
	} else {
		win.webViews[0].LoadURI(uri)
	}

	// Due to a bug with keypresses registering multiple times, we ignore
	// keypresses within 10ms of each other.
	// After each keypress, true gets sent to this channel 10ms after.
	win.timeoutChan <- true

	handle, err := win.Window.Window.Connect("key-press-event", win.handleKeyPress)
	if err == nil {
		win.windowSignalHandles = append(
			win.windowSignalHandles,
			&signalHandle{win.Window.Window.Object, handle})
	}
	handle, err = win.Window.Window.Connect(
		"button-press-event",
		win.handleBackForwardButtons)
	if err == nil {
		win.windowSignalHandles = append(
			win.windowSignalHandles,
			&signalHandle{win.Window.Window.Object, handle})
	}
	// handle middle click primary selection paste.
	handle, err = win.Window.StatusBar.Container.Connect("button-press-event",
		func(_ interface{}, e *gdk.Event) bool {
			bpe := (*C.GdkEventButton)(unsafe.Pointer(e.Native()))
			if bpe.button != 2 {
				return false
			}
			cmdState, ok := win.State.(*cmd.CommandLineMode)
			if !ok {
				return false
			}
			clip, err := gtk.ClipboardGet(gdk.SELECTION_PRIMARY)
			if err != nil {
				win.logErrorf("Failed to acquire clipboard: %v", err)
				return false
			}
			str, err := clip.WaitForText()
			if err != nil {
				return true
			}
			win.setState(cmdState.Paste(str))
			return true
		})
	if err == nil {
		win.windowSignalHandles = append(
			win.windowSignalHandles,
			&signalHandle{win.Window.StatusBar.Container.Object, handle})
	}
	handle, err = win.Window.Window.Connect("destroy", func() {
		for _, wv := range win.webViews {
			wv.close()
		}
		for _, h := range win.activeSignalHandles {
			h.disconnect()
		}
		for _, h := range win.windowSignalHandles {
			h.disconnect()
		}
		g.closeWindow(win)
		// Ensure garbage collection
		win.Window.WebView = nil
		win.bindings = nil
		win.builtins = nil
		win.State = nil
		schedGc()
	})
	if err == nil {
		win.windowSignalHandles = append(
			win.windowSignalHandles,
			&signalHandle{win.Window.Window.Object, handle})
	}

	win.Show()
	return win, nil
}

// handleBackForwardButtons handles the back / forward mouse button presses.
func (w *Window) handleBackForwardButtons(_ interface{}, e *gdk.Event) bool {
	bpe := (*C.GdkEventButton)(unsafe.Pointer(e.Native()))
	wv := w.getWebView()
	switch bpe.button {
	// Back button
	case 8:
		if wv.CanGoBack() {
			wv.GoBack()
			return true
		} else {
			return false
		}
	// Forward button
	case 9:
		if wv.CanGoForward() {
			wv.GoForward()
			return true
		} else {
			return false
		}
	default:
		return false
	}
}

// handleKeyPress handles a gdk key press event.
func (w *Window) handleKeyPress(uiWin *gtk.Window, e *gdk.Event) bool {
	select {
	case <-w.timeoutChan:
		// Make sure that the timeout is properly applied.
		defer func() {
			go func() {
				<-time.After(keyTimeout)
				w.timeoutChan <- true
			}()
		}()
		// This conversion *shouldn't* be unsafe, BUT we really don't want
		// crashes here. TODO
		ek := gdk.EventKey{e}
		key := cmd.NewKeyFromEventKey(ek)
		if PrintKeys {
			log.Printf("%v", key)
		}
		// We ignore modifier keys.
		if key.IsModifier {
			return false
		}

		oldState := w.State
		newState, ret := w.State.ProcessKeyPress(key)
		// If this is not the case, a state change command was issued. This
		// takes precedence.
		if oldState == w.State {
			w.setState(newState)
		} else if statusM, ok := w.State.(*cmd.StatusMode); ok && statusM.State == oldState {
			w.setState(cmd.NewStatusMode(newState, statusM.Substate, statusM.Status))
		} else if confM, ok := w.State.(*cmd.ConfirmMode); ok && confM.State == oldState {
			w.setState(&cmd.ConfirmMode{
				newState,
				confM.Substate,
				confM.Prompt,
				confM.ConfirmKeys,
				confM.CancelKeys,
				confM.Default,
				confM.Callback,
			})
		}
		return ret
	default:
		return false
	}
}

// rebuildBindings rebuilds the bindings for this window.
func (w *Window) rebuildBindings() {
	bindings, errs := cmd.ParseRawBindings(w.parent.rawBindings, w.builtins, w.runCmd)
	if errs != nil {
		for _, err := range errs {
			w.logErrorf("Error: Failed to parse key bindings: %v", err)
		}
		(*Window)(nil).logError("Faulty bindings have been dropped.")
	}
	bindingTree, errs := cmd.NewBindingTree(bindings)
	if errs != nil {
		for _, err := range errs {
			w.logErrorf("Error: Failed to parse key bindings: %v", err)
		}
		(*Window)(nil).logError("Faulty bindings have been dropped.")
	}
	w.bindings[states.NormalSubstateNormal] = bindingTree
}

// rebuildQuickmarks rebuild the quickmark bindings for this window.
func (w *Window) rebuildQuickmarks() {
	bindings := make([]*cmd.Binding, 0, len(w.parent.quickmarks))
	for keyStr, _ := range w.parent.quickmarks {
		bindings = append(
			bindings,
			&cmd.Binding{cmd.ParseKeys(keyStr), w.quickmarkCallback})
	}
	bindingTree, errs := cmd.NewBindingTree(bindings)
	if errs != nil {
		for _, err := range errs {
			w.logErrorf("Error: Failed to parse quickmarks: %v", err)
		}
		(*Window)(nil).logError("Faulty quickmarks have been dropped.")
	}
	w.bindings[states.NormalSubstateQuickmark] = bindingTree
	w.bindings[states.NormalSubstateQuickmarkTab] = bindingTree
	w.bindings[states.NormalSubstateQuickmarkWindow] = bindingTree
	w.bindings[states.NormalSubstateQuickmarksRapid] = bindingTree
}

// quickmarkCallback opens a quickmark as a callback from a binding execution.
func (w *Window) quickmarkCallback(keys []cmd.Key, _ *int, s cmd.Substate) {
	uri, ok := w.parent.quickmarks[cmd.KeysString(keys)]
	if !ok {
		w.logErrorf("Unknown quickmark: %s", cmd.KeysString(keys))
		return
	}
	switch s {
	case states.NormalSubstateQuickmark:
		w.getWebView().LoadURI(uri)
	case states.NormalSubstateQuickmarkTab:
		w.NewTab(uri)
		w.tabNext()
	case states.NormalSubstateQuickmarkWindow:
		w.parent.NewWindow(uri)
	case states.NormalSubstateQuickmarksRapid:
		w.NewTab(uri)
		w.setState(cmd.NewNormalModeWithSubstate(
			w.State,
			states.NormalSubstateQuickmarksRapid))
	default:
		w.logErrorf("Quickmark opened from non-quickmark substate: %d", s)
		return
	}
}

// getWebView retrieves the currently active webView.
func (w *Window) getWebView() *webView {
	return w.webViews[w.currentWebView]
}

// reconnectWebViewSignals switches the connected signals from the old web
// view (if any) to the currently connected one.
func (w *Window) reconnectWebViewSignals() {
	for _, handle := range w.activeSignalHandles {
		handle.disconnect()
	}

	wv := w.getWebView().WebView

	w.activeSignalHandles = make([]*signalHandle, 0, 6)

	titleSetFunc := func() {
		title := wv.GetTitle()
		if title != "" {
			w.SetTitle(fmt.Sprintf("%s - Golem", title))
		} else {
			w.SetTitle("Golem")
		}
	}
	titleSetFunc()

	handle, err := wv.Connect("notify::title", titleSetFunc)
	w.activeSignalHandles = append(
		w.activeSignalHandles,
		newSignalHandle(wv.Object, handle, err))

	handle, err = wv.Connect("notify::uri", w.UpdateLocation)
	w.activeSignalHandles = append(
		w.activeSignalHandles,
		newSignalHandle(wv.Object, handle, err))

	handle, err = wv.Connect("notify::estimated-load-progress", w.UpdateLocation)
	w.activeSignalHandles = append(
		w.activeSignalHandles,
		newSignalHandle(wv.Object, handle, err))

	bfl := wv.GetBackForwardList()
	handle, err = bfl.Connect("changed", w.UpdateLocation)
	w.activeSignalHandles = append(
		w.activeSignalHandles,
		newSignalHandle(bfl.Object, handle, err))

	handle, err = wv.Connect("enter-fullscreen", w.Window.HideUI)
	w.activeSignalHandles = append(
		w.activeSignalHandles,
		newSignalHandle(wv.Object, handle, err))

	handle, err = wv.Connect("leave-fullscreen", w.Window.ShowUI)
	w.activeSignalHandles = append(
		w.activeSignalHandles,
		newSignalHandle(wv.Object, handle, err))
}

// runCmd runs a command.
func (w *Window) runCmd(cmd string) {
	runCmd(w, w.parent, cmd)
}

// runCmd runs a command.
func runCmd(w *Window, g *Golem, command string) {
	// Space followed optionally by a line comment (starting with ")
	if blankLineRegex.MatchString(command) {
		return
	}

	parts, err := shellwords.Parse(command)
	if err != nil {
		w.logErrorf("Error: Failed to parse command '%v': %v", command, err)
		return
	}
	if len(parts[0]) == 0 {
		parts = parts[1:len(parts)]
	}
	f, ok := commands[parts[0]]
	if ok {
		if PrintCommands {
			log.Printf("Running command '%v'.", command)
		}
		f(w, g, parts)
	} else {
		w.logErrorf("Error: Failed to run command '%v': No such command.", command)
	}
}

// addDownload adds an active download.
func (w *Window) addDownload(d *webkit.Download) {
	w.setState(cmd.NewStatusMode(
		w.State,
		states.StatusSubstateMajor,
		"Download started..."))
}

// logError logs (and displays) an error message.
func (w *Window) logError(err string) {
	if w != nil {
		w.setState(cmd.NewStatusMode(
			w.State,
			states.StatusSubstateError,
			err))
	}
	Errlog.Println(err)
}

// logErrorf logs (and displays) an errormessage, supplies as a format string
// with arguments.
func (w *Window) logErrorf(fmtStr string, args ...interface{}) {
	str := fmt.Sprintf(fmtStr, args...)
	w.logError(str)
}
