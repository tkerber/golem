package ui

import (
	"fmt"
	"html"
	"math"
	"unicode/utf8"

	"github.com/conformal/gotk3/gtk"
	"github.com/conformal/gotk3/pango"
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

	tabBar := &TabBar{box, make([]*TabBarTab, 0, 100), parent, nil, "", 0, "", 0}

	return tabBar, nil
}

// AppendTab creates a new TabBarTab and appends it to the TabBar.
func (tb *TabBar) AppendTab() (*TabBarTab, error) {
	tab, err := newTabBarTab(tb, len(tb.tabs))
	if err != nil {
		return nil, err
	}
	GlibMainContextInvoke(func() {
		tb.tabs = append(tb.tabs, tab)
		tb.PackStart(tab.Label, false, false, 0)
		tab.Label.Show()
		tab.redraw()
	})

	tb.UpdateFormatString()
	return tab, nil
}

// AddTab creates a new TabBarTab and adds it into a specified position.
func (tb *TabBar) AddTab(pos int) (*TabBarTab, error) {
	tab, err := tb.AppendTab()
	if err != nil {
		return nil, err
	}
	tb.MoveTab(pos, len(tb.tabs)-1)
	return tab, nil
}

// UpdateFormatString checks if the format string is still valid, and if not,
// updates it and redraws all tabs.
func (tb *TabBar) UpdateFormatString() {
	GlibMainContextInvoke(func() {
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

// MoveTab moves a tab from a specified index to another.
func (tb *TabBar) MoveTab(to, from int) {
	GlibMainContextInvoke(func() {
		tab := tb.tabs[from]
		tb.ReorderChild(tab.Label, to)
		if from > to {
			// move tabs further along tabs array.
			copy(
				tb.tabs[to+1:from+1],
				tb.tabs[to:from])
			// insert tab
			tb.tabs[to] = tab
			// redraw all affected tabs
			for i, t := range tb.tabs[to : from+1] {
				t.index = to + i
				t.redraw()
			}
		} else if to > from {
			// move tabs back along tabs array.
			copy(
				tb.tabs[from:to],
				tb.tabs[from+1:to+1])
			// insert tab
			tb.tabs[to] = tab
			// redraw all affected tabs
			for i, t := range tb.tabs[from : to+1] {
				t.index = from + i
				t.redraw()
			}
		}
	})
}

// FocusTab focuses the tab at the given index.
//
// Any currently focused tab is unfocused.
func (tb *TabBar) FocusTab(i int) {
	GlibMainContextInvoke(func() {
		if tb.focused != nil {
			tb.focused.focused = false
			tb.focused.redraw()
		}
		tb.tabs[i].focused = true
		tb.tabs[i].redraw()
		tb.focused = tb.tabs[i]
	})
}

// PopTab removes the last tab.
func (tb *TabBar) PopTab() {
	GlibMainContextInvoke(func() {
		tab := tb.tabs[len(tb.tabs)-1]
		tb.Remove(tab)
		tb.tabs = tb.tabs[:len(tb.tabs)-1]
		if tab == tb.focused {
			tb.focused = nil
		}
	})

	tb.UpdateFormatString()
}

// CloseTab removes the tab at the given index.
func (tb *TabBar) CloseTab(i int) {
	tb.MoveTab(len(tb.tabs)-1, i)
	tb.PopTab()
}

// A TabBarTab is the display of a single tab name.
type TabBarTab struct {
	*gtk.Label
	parent       *TabBar
	title        string
	index        int
	focused      bool
	loadProgress float64
}

// newTabBarTab creates a new TabBarTab in a given TabBar at a given index.
func newTabBarTab(parent *TabBar, index int) (*TabBarTab, error) {
	l, err := gtk.LabelNew("")
	if err != nil {
		return nil, err
	}
	l.SetEllipsize(pango.ELLIPSIZE_END)
	l.OverrideFont("monospace")
	// Seriously? Is this the only way to limit the width?
	l.SetMaxWidthChars(15)
	l.SetWidthChars(15)
	l.SetUseMarkup(true)
	return &TabBarTab{l, parent, "", index, false, 1.0}, nil
}

// SetTitle sets the tabs title.
func (t *TabBarTab) SetTitle(title string) {
	t.title = title
	GlibMainContextInvoke(t.redraw)
}

// SetLoadProgress sets the load progress to be displayed for this tab.
func (t *TabBarTab) SetLoadProgress(to float64) {
	t.loadProgress = to
	GlibMainContextInvoke(t.redraw)
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
	t.SetMarkup(t.parent.parent.MarkupReplacer.Replace(text))
}
