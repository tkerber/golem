package ui

// #cgo pkg-config: gdk-3.0
// #include <gdk/gdk.h>
import "C"
import (
	"fmt"
	"html"
	"math"
	"unicode/utf8"
	"unsafe"

	"github.com/conformal/gotk3/gdk"
	"github.com/conformal/gotk3/glib"
	"github.com/conformal/gotk3/gtk"
	"github.com/conformal/gotk3/pango"
	ggtk "github.com/tkerber/golem/gtk"
)

// A TabBar is a bar containing tab displays.
type TabBar struct {
	*gtk.Box
	tabs                 []*TabBarTab
	parent               *Window
	focused              *TabBarTab
	fmtString            string
	fmtStringBaseLen     int
	fmtLoadString        string
	fmtLoadStringBaseLen int
}

// NewTabBar creates a new TabBar for a Window.
func NewTabBar(parent *Window) (*TabBar, error) {
	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 1)
	if err != nil {
		return nil, err
	}
	box.SetName("tabbar")

	tabBar := &TabBar{
		box,
		make([]*TabBarTab, 0, 100),
		parent,
		nil,
		"",
		0,
		"",
		0}

	return tabBar, nil
}

// AppendTab is a wrapper around appendTabs which appends a single TabBarTab
// to the TabBar.
func (tb *TabBar) AppendTab() (*TabBarTab, error) {
	tabs, err := tb.appendTabs(1)
	if err != nil {
		return nil, err
	}
	return tabs[0], nil
}

// appendTabs creates a new sequence of TabBarTabs and appends them to the
// TabBar.
func (tb *TabBar) appendTabs(n int) ([]*TabBarTab, error) {
	tabs := make([]*TabBarTab, n)
	for i := 0; i < n; i++ {
		tab, err := newTabBarTab(tb, len(tb.tabs)+i)
		if err != nil {
			return nil, err
		}
		tabs[i] = tab
	}
	ggtk.GlibMainContextInvoke(func() {
		for _, tab := range tabs {
			tb.tabs = append(tb.tabs, tab)
			tb.PackStart(tab.EventBox, false, false, 0)
			tab.EventBox.ShowAll()
			tab.redraw()
		}
	})

	tb.UpdateFormatString()
	return tabs, nil
}

// AddTab is a wrapper around AddTabs which inserts a single tab at the
// specified index.
func (tb *TabBar) AddTab(i int) (*TabBarTab, error) {
	tabs, err := tb.AddTabs(i, i+1)
	if err != nil {
		return nil, err
	}
	return tabs[0], nil
}

// AddTabs creates a new sequence of tabs between two indicies.
func (tb *TabBar) AddTabs(i, j int) ([]*TabBarTab, error) {
	tabs, err := tb.appendTabs(j - i)
	if err != nil {
		return nil, err
	}
	tb.moveTabs(i, len(tb.tabs)-(j-i), len(tb.tabs))
	return tabs, nil
}

// UpdateFormatString checks if the format string is still valid, and if not,
// updates it and redraws all tabs.
func (tb *TabBar) UpdateFormatString() {
	ggtk.GlibMainContextInvoke(func() {
		var numLen int
		numLen = int(math.Floor(math.Log10(float64(len(tb.tabs))))) + 1
		fmtString := fmt.Sprintf("<num>%%0%dd</num> %%s", numLen)
		fmtLoadString := fmt.Sprintf(
			"<num>%%0%dd</num> [<load>%%02d%%%%</load>] %%s", numLen)
		if tb.fmtString != fmtString || tb.fmtLoadString != fmtLoadString {
			tb.fmtString = fmtString
			tb.fmtLoadString = fmtLoadString
			tb.fmtStringBaseLen = numLen + 1
			tb.fmtLoadStringBaseLen = numLen + 7
			// redraw all tabs
			for i, t := range tb.tabs {
				t.index = i
				t.redraw()
			}
		}
	})
}

// moveTabs moves a block of tabs from one space to another.
func (tb *TabBar) moveTabs(toStart, fromStart, fromEnd int) {
	ggtk.GlibMainContextInvoke(func() {
		toEnd := toStart + (fromEnd - fromStart)
		tabs := make([]*TabBarTab, fromEnd-fromStart)
		copy(tabs, tb.tabs[fromStart:fromEnd])
		if toStart > fromStart {
			for i := len(tabs) - 1; i >= 0; i-- {
				tb.ReorderChild(tabs[i].EventBox, i+toStart)
			}
			// move tabs back along tabs array.
			copy(
				tb.tabs[fromStart:toStart],
				tb.tabs[fromEnd:toEnd])
			// insert tabs
			copy(tb.tabs[toStart:toEnd], tabs)
			// redraw all affected tabs
			for i, t := range tb.tabs[fromStart:toEnd] {
				t.index = fromStart + i
				t.redraw()
			}
		} else if toStart < fromStart {
			for i, tab := range tabs {
				tb.ReorderChild(tab.EventBox, i+toStart)
			}
			// move tas further along tabs array.
			copy(
				tb.tabs[toEnd:fromEnd],
				tb.tabs[toStart:fromStart])
			// insert tabs
			copy(tb.tabs[toStart:toEnd], tabs)
			// redraw all affected tabs
			for i, t := range tb.tabs[toStart:fromEnd] {
				t.index = toStart + i
				t.redraw()
			}
		}
	})
}

// FocusTab focuses the tab at the given index.
//
// Any currently focused tab is unfocused.
func (tb *TabBar) FocusTab(i int) {
	ggtk.GlibMainContextInvoke(func() {
		if tb.focused != nil {
			tb.focused.focused = false
			tb.focused.redraw()
		}
		tb.tabs[i].focused = true
		tb.tabs[i].redraw()
		tb.focused = tb.tabs[i]
	})
}

// popTabs removes the n tabs
func (tb *TabBar) popTabs(n int) {
	ggtk.GlibMainContextInvoke(func() {
		delSlice := tb.tabs[len(tb.tabs)-n:]
		for i, tab := range delSlice {
			tb.Remove(tab)
			if tab == tb.focused {
				tb.focused = nil
			}
			tab.Free()
			delSlice[i] = nil
		}
		tb.tabs = tb.tabs[:len(tb.tabs)-n]
	})

	tb.UpdateFormatString()
}

// CloseTabs removes the tabs between the given indicies (slice indexes)
func (tb *TabBar) CloseTabs(i, j int) {
	tb.moveTabs(len(tb.tabs)-(j-i), i, j)
	tb.popTabs(j - i)
}

// A TabBarTab is the display of a single tab name.
type TabBarTab struct {
	*gtk.EventBox
	label        *gtk.Label
	parent       *TabBar
	title        string
	index        int
	focused      bool
	loadProgress float64
	handles      []glib.SignalHandle
}

// newTabBarTab creates a new TabBarTab in a given TabBar at a given index.
func newTabBarTab(parent *TabBar, index int) (*TabBarTab, error) {
	box, err := gtk.EventBoxNew()
	if err != nil {
		return nil, err
	}
	l, err := gtk.LabelNew("")
	if err != nil {
		return nil, err
	}
	box.Add(l)
	l.SetEllipsize(pango.ELLIPSIZE_END)
	l.OverrideFont("monospace")
	// Seriously? Is this the only way to limit the width?
	l.SetMaxWidthChars(15)
	l.SetWidthChars(15)
	l.SetUseMarkup(true)
	t := &TabBarTab{
		box,
		l,
		parent,
		"",
		index,
		false,
		1.0,
		make([]glib.SignalHandle, 0, 5),
	}
	handle, err := box.Connect("button-press-event",
		func(_ interface{}, e *gdk.Event) bool {
			bpe := (*C.GdkEventButton)(unsafe.Pointer(e.Native()))
			if bpe.button != 1 {
				return false
			}
			t.parent.parent.TabGo(t.index)
			return true
		})
	if err == nil {
		t.handles = append(t.handles, handle)
	}
	return t, nil
}

// SetTitle sets the tabs title.
func (t *TabBarTab) SetTitle(title string) {
	t.title = title
	ggtk.GlibMainContextInvoke(t.redraw)
}

// SetLoadProgress sets the load progress to be displayed for this tab.
func (t *TabBarTab) SetLoadProgress(to float64) {
	t.loadProgress = to
	ggtk.GlibMainContextInvoke(t.redraw)
}

// Free ensures that all connections which could keep the tab from being GC'd
// are broken.
func (t *TabBarTab) Free() {
	for _, handle := range t.handles {
		t.EventBox.HandlerDisconnect(handle)
	}
}

// redraw redraws the tab.
//
// Should only be invoked in glib's main context.
func (t *TabBarTab) redraw() {
	title := t.title
	if title == "" {
		title = "[untitled]"
	}
	var text string
	var length int
	if t.loadProgress == 1.0 {
		text = fmt.Sprintf(
			t.parent.fmtString,
			t.index+1,
			html.EscapeString(title))
		// Pad to short text. God this stuff is horrible code...
		length = t.parent.fmtStringBaseLen + utf8.RuneCountInString(title)
	} else {
		text = fmt.Sprintf(
			t.parent.fmtLoadString,
			t.index+1,
			int(t.loadProgress*100),
			html.EscapeString(title))
		length = t.parent.fmtLoadStringBaseLen + utf8.RuneCountInString(title)
	}
	for i := length; i < 15; i++ {
		text += " "
	}
	if t.focused {
		text = fmt.Sprintf("<focus>%s</focus>", text)
	}
	t.label.SetMarkup(t.parent.parent.MarkupReplacer.Replace(text))
}
