package golem

import (
	"errors"
	"log"
	"math"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"

	"github.com/mattn/go-shellwords"
	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/golem/states"
)

const HintsChars = "FDSARTGBVECWXQZIOPMNHYULKJ"

type nothing struct{}

// A RPCSession manages listening on golems RPC socket, as well as serving
// as the exported RPC object for golem.
type RPCSession struct {
	listener net.Listener
	closed   bool
	golem    *Golem
}

func NewRPCSession(l net.Listener) *RPCSession {
	s := &RPCSession{l, false, nil}
	go func() {
		for {
			c, err := s.listener.Accept()
			if err != nil && !s.closed {
				log.Printf(
					"Failed to accept new connection on socket: %v", err)
				continue
			} else if err != nil && s.closed {
				return
			}
			// serve the new connection.
			go func() {
				client := jsonrpc.NewClient(c)
				var id uint64
				client.Call("GolemWebExtension.GetPageID", nil, &id)
				wv := s.golem.webViews[id]
				wv.webExtension.conn = c
				wv.webExtension.client = client
				jsonrpc.ServeConn(c)
			}()
		}
	}()
	return s
}

// NewWindow creates a new window in golem's main process.
func (s *RPCSession) NewWindow(args *nothing, ret *nothing) error {
	_, err := s.golem.NewWindow("")
	return err
}

// NewTabs opens a set of uris in new tabs.
func (s *RPCSession) NewTabs(uris []string, ret *nothing) error {
	// we try to split it into parts to allow searches to be passed
	// via command line. If this fails, we ignore the error and just
	// pass the whole string instead.
	for i, uri := range uris {
		parts, err := shellwords.Parse(uri)
		if err != nil {
			parts = []string{uri}
		}
		uris[i] = s.golem.OpenURI(parts)
	}
	w := s.golem.windows[0]
	_, err := w.NewTabs(uris...)
	if err != nil {
		return err
	}
	w.TabNext()
	return nil
}

// A BlockQuery encapsulates all the arguments for querying the blocked
// status of a website.
type BlockQuery struct {
	Uri        string
	FirstParty string
	Flags      uint64
}

// Blocks checks whether a uri is blocked by the adblocker or not.
func (s *RPCSession) Blocks(bq BlockQuery, ret *bool) error {
	*ret = s.golem.adblocker.Blocks(bq.Uri, bq.FirstParty, bq.Flags)
	return nil
}

// DomainElemHideCSS retrieves the css string to hide the elements on a given
// domain.
func (s *RPCSession) DomainElemHideCSS(domain string, ret *string) error {
	*ret = s.golem.adblocker.DomainElemHideCSS(domain)
	return nil
}

// GetHintsLabels gets n labels for hints.
func (s *RPCSession) GetHintsLabels(n int64, ret *[]string) error {
	*ret = make([]string, n)
	if n == 0 {
		return nil
	}
	length := int(math.Ceil(
		math.Log(float64(n)) / math.Log(float64(len(HintsChars)))))
	for i := range *ret {
		bytes := make([]byte, length)
		divI := i
		for j := range bytes {
			bytes[j] = HintsChars[divI%len(HintsChars)]
			divI /= len(HintsChars)
		}
		(*ret)[i] = string(bytes)
	}
	return nil
}

// A HintCallRequest is a request to call a particular hint from a given web
// view.
type HintCallRequest struct {
	Id  uint64
	Uri string
}

// HintCall is called if a hint was hit.
func (g *RPCSession) HintCall(hcr HintCallRequest, ret *bool) error {
	wv, ok := g.golem.webViews[hcr.Id]
	if !ok {
		return errors.New("Invalid web page id recieved.")
	}
	w := wv.window
	if w == nil {
		return errors.New("WebView is not attached to any window.")
	}
	hm, ok := w.State.(*states.HintsMode)
	if !ok {
		return errors.New("Window not currently in hints mode.")
	}
	*ret = hm.ExecuterFunction(hcr.Uri)
	if *ret == true {
		w.setState(&states.HintsMode{
			hm.StateIndependant,
			hm.Substate,
			hm.HintsCallback,
			nil,
			hm.ExecuterFunction,
		})
	}
	return nil
}

// A VerticalPositionChange records the change in the vertical position of a
// particular web view.
type VerticalPositionChange struct {
	Id     uint64
	Top    int64
	Height int64
}

// VerticalPositionChanged is called to signal a change in the vertical
// position of a web page.
func (g *RPCSession) VerticalPositionChanged(
	vpc VerticalPositionChange,
	ret *nothing) error {

	wv, ok := g.golem.webViews[vpc.Id]
	if !ok {
		return errors.New("Invalid web page id recieved.")
	}
	wv.top = vpc.Top
	wv.height = vpc.Height
	for _, w := range g.golem.windows {
		if wv == w.getWebView() {
			w.UpdateLocation()
		}
	}
	return nil
}

type InputFocusChange struct {
	Id      uint64
	Focused bool
}

// InputFocusChanged is called to signal a change in the input focus of a web
// page.
func (g *RPCSession) InputFocusChanged(
	ifc InputFocusChange, ret *nothing) error {

	wv, ok := g.golem.webViews[ifc.Id]
	if !ok {
		return errors.New("Invalid web page id recieved.")
	}
	// If it's newly focused, set any windows with this webview
	// displayed to insert mode.
	//
	// Otherwise, if the window is currently in insert mode and it's
	// newly unfocused, set this webview to normal mode.
	for _, w := range g.golem.windows {
		if wv == w.getWebView() {
			if ifc.Focused {
				w.setState(
					cmd.NewInsertMode(w.State, cmd.SubstateDefault))
			} else if _, ok := w.State.(*cmd.InsertMode); ok {
				w.setState(
					cmd.NewNormalMode(w.State))
			}
		}
	}
	return nil
}

type webExtension struct {
	conn   net.Conn
	client *rpc.Client
}

func (w *webExtension) call(
	method string,
	args interface{},
	reply interface{}) error {

	if w.client == nil {
		return errors.New("Failed RPC call: web view not connected.")
	}
	w.client.Call(method, args, reply)
	return nil
}

// LinkHintsMode initializes hints mode for links.
func (w *webExtension) LinkHintsMode() (int64, error) {
	var ret int64
	err := w.call("GolemWebExtension.LinkHintsMode", nil, &ret)
	return ret, err
}

// FormVariableHintsMode initializes hints mode for form input fields.
func (w *webExtension) FormVariableHintsMode() (int64, error) {
	var ret int64
	err := w.call("GolemWebExtension.FormVariableHintsMode", nil, &ret)
	return ret, err
}

// ClickHintsMode initializes hints mode for clickable elements.
func (w *webExtension) ClickHintsMode() (int64, error) {
	var ret int64
	err := w.call("GolemWebExtension.ClickHintsMode", nil, &ret)
	return ret, err
}

// EndHintsMode ends hints mode.
func (w *webExtension) EndHintsMode() error {
	return w.call("GolemWebExtension.EndHintsMode", nil, nil)
}

// FilterHintsMode filters the displayed hints in hints mode.
//
// If a hint is matched precicely by a filter, it is hit.
func (w *webExtension) FilterHintsMode(filter string) (bool, error) {
	var ret bool
	err := w.call("GolemWebExtension.FilterHintsMode", filter, &ret)
	return ret, err
}

// getInt64 retrieves an int64 value.
func (w *webExtension) getInt64(name string) (int64, error) {
	var ret int64
	err := w.call("GolemWebExtension.Get", name, &ret)
	return ret, err
}

// getScrollTop retrieves the webExtension's scroll position from the top of
// the page.
func (w *webExtension) getScrollTop() (int64, error) {
	return w.getInt64("ScrollTop")
}

// getScrollLeft retrieves the webExtension's scroll position from the left of
// the page.
func (w *webExtension) getScrollLeft() (int64, error) {
	return w.getInt64("ScrollLeft")
}

// getScrollWidth retrieves the webExtension's scroll area width.
func (w *webExtension) getScrollWidth() (int64, error) {
	return w.getInt64("ScrollWidth")
}

// getScrollHeight retrieves the webExtension's scroll area height.
func (w *webExtension) getScrollHeight() (int64, error) {
	return w.getInt64("ScrollHeight")
}

// getScrollTargetTop retrieves the webExtension's scroll position from the
// top of the target scroll area.
func (w *webExtension) getScrollTargetTop() (int64, error) {
	return w.getInt64("ScrollTargetTop")
}

// getScrollTargetLeft retrieves the webExtension's scroll position from the
// left of the target scroll area.
func (w *webExtension) getScrollTargetLeft() (int64, error) {
	return w.getInt64("ScrollTargetLeft")
}

// getScrollTargetWidth retrieves the webExtension's target scroll area width.
func (w *webExtension) getScrollTargetWidth() (int64, error) {
	return w.getInt64("ScrollTargetWidth")
}

// getScrollTargetHeight retrieves the webExtension's target scroll area height.
func (w *webExtension) getScrollTargetHeight() (int64, error) {
	return w.getInt64("ScrollTargetHeight")
}

type Int64SetInstruction struct {
	Name  string
	Value int64
}

// setInf64 sets an int64 value.
func (w *webExtension) setInt64(name string, to int64) error {
	return w.call("GolemWebExtension.Set", Int64SetInstruction{name, to}, nil)
}

// setScrollTop sets the webExtension's scroll position from the top.
func (w *webExtension) setScrollTop(to int64) error {
	return w.setInt64("ScrollTop", to)
}

// setScrollLeft sets the webExtension's scroll position from the left.
func (w *webExtension) setScrollLeft(to int64) error {
	return w.setInt64("ScrollLeft", to)
}

// setScrollTargetTop sets the webExtension's scroll position from the top of
// the target scroll area..
func (w *webExtension) setScrollTargetTop(to int64) error {
	return w.setInt64("ScrollTargetTop", to)
}

// setScrollTargetLeft sets the webExtension's scroll position from the left
// of the target scroll area.
func (w *webExtension) setScrollTargetLeft(to int64) error {
	return w.setInt64("ScrollTargetLeft", to)
}
