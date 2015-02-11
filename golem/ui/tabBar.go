package ui

// #cgo pkg-config: gdk-3.0
// #cgo pkg-config: gtk+-3.0
// #cgo pkg-config: cairo
// #include <gdk/gdk.h>
// #include <gtk/gtk.h>
// #include <cairo.h>
/*

// cairoSize retrieves the size of a cairo surface.
static void inline
cairoSize(cairo_surface_t *s, double *width, double *height) {
	double x1, x2, y1, y2;
	cairo_t *cr = cairo_create(s);
	cairo_clip_extents(cr, &x1, &y1, &x2, &y2);
	cairo_destroy(cr);
	*width = x2 - x1;
	*height = y2 - y1;
}

// scaleSurface scales a cairo surface to the given dimensions.
static cairo_surface_t *
scaleSurface(cairo_surface_t *s, double width, double height) {
	double oldWidth, oldHeight;
	cairoSize(s, &oldWidth, &oldHeight);
	cairo_surface_t *ret = cairo_surface_create_similar(
		s, CAIRO_CONTENT_COLOR_ALPHA, (int)width, (int)height);
	cairo_t *cr = cairo_create(ret);
	cairo_scale(cr, width/oldWidth, height/oldHeight);
	cairo_set_source_surface(cr, s, 0, 0);
	cairo_pattern_set_extend(cairo_get_source(cr), CAIRO_EXTEND_PAD);
	cairo_set_operator(cr, CAIRO_OPERATOR_SOURCE);
	cairo_paint(cr);
	cairo_destroy(cr);
	return ret;
}
*/
import "C"
import (
	"fmt"
	"html"
	"math"
	"runtime"
	"unsafe"

	"github.com/conformal/gotk3/gdk"
	"github.com/conformal/gotk3/glib"
	"github.com/conformal/gotk3/gtk"
	"github.com/conformal/gotk3/pango"
	ggtk "github.com/tkerber/golem/gtk"
)

// A TabBar is a bar containing tab displays.
type TabBar struct {
	*gtk.ScrolledWindow
	ebox                 *gtk.EventBox
	box                  *gtk.Box
	tabs                 []*TabBarTab
	parent               *Window
	focused              *TabBarTab
	fmtString            string
	fmtStringBaseLen     int
	fmtLoadString        string
	fmtLoadStringBaseLen int
	handles              []glib.SignalHandle
}

// TabBarSpacing is the spacing between individual tabs in the tab bar.
const TabBarSpacing = 1

// NewTabBar creates a new TabBar for a Window.
func NewTabBar(parent *Window) (*TabBar, error) {
	scrollWin, err := gtk.ScrolledWindowNew(nil, nil)
	if err != nil {
		return nil, err
	}
	scrollWin.SetSizeRequest(120, -1)
	cScrollWin := (*C.GtkScrolledWindow)(unsafe.Pointer(scrollWin.Native()))
	cScrollbar := C.gtk_scrolled_window_get_hscrollbar(cScrollWin)
	C.gtk_widget_hide(cScrollbar)
	C.gtk_widget_set_no_show_all(cScrollbar, C.TRUE)
	cScrollbar = C.gtk_scrolled_window_get_vscrollbar(cScrollWin)
	C.gtk_widget_hide(cScrollbar)
	C.gtk_widget_set_no_show_all(cScrollbar, C.TRUE)

	ebox, err := gtk.EventBoxNew()
	if err != nil {
		return nil, err
	}
	ebox.AddEvents(C.GDK_SCROLL_MASK)
	scrollWin.Add(ebox)

	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, TabBarSpacing)
	if err != nil {
		return nil, err
	}
	box.SetName("tabbar")
	ebox.Add(box)

	tabBar := &TabBar{
		scrollWin,
		ebox,
		box,
		make([]*TabBarTab, 0, 100),
		parent,
		nil,
		"",
		0,
		"",
		0,
		make([]glib.SignalHandle, 0, 5),
	}

	winHandles := make([]glib.SignalHandle, 2)
	winHandles[0], err = scrollWin.Connect("size-allocate", tabBar.reposition)
	if err != nil {
		return nil, err
	}
	winHandles[1], err = scrollWin.Connect("destroy", func() {
		for _, h := range winHandles {
			scrollWin.HandlerDisconnect(h)
		}
	})

	// Scroll tabs up/down.
	handle, err := ebox.Connect("scroll-event",
		func(_ interface{}, e *gdk.Event) {
			se := (*C.GdkEventScroll)(unsafe.Pointer(e.Native()))
			switch se.direction {
			case C.GDK_SCROLL_UP:
				tabBar.parent.TabPrev()
			case C.GDK_SCROLL_DOWN:
				tabBar.parent.TabNext()
			}
		})
	if err == nil {
		tabBar.handles = append(tabBar.handles, handle)
	}

	return tabBar, nil
}

// reposition ensures that the right tabs are currently shown on screen.
func (tb *TabBar) reposition() {
	allocHeight := tb.ScrolledWindow.GetAllocatedHeight()
	// Find active tab
	var i int
	for i = range tb.tabs {
		if tb.tabs[i] == tb.focused {
			break
		}
	}
	tb.tabs[i].Show()
	allocHeight -= tb.tabs[i].GetAllocatedHeight()
	show := true
	// Look through all tabs, from the focused one outwards.
	// Show all of them, until the allocated height runs out. Then, switch to
	// hide mode and hide the rest.
	for j := 1; j <= i || j <= len(tb.tabs)-i; j++ {
		for _, mult := range []int{-1, 1} {
			if (mult == -1 && i-j < 0) || (mult == 1 && i+j >= len(tb.tabs)) {
				continue
			}
			index := i + mult*j
			if show {
				tb.tabs[index].Show()
				h := tb.tabs[index].GetAllocatedHeight()
				h += TabBarSpacing
				if h <= allocHeight {
					allocHeight -= h
				} else {
					show = false
				}
			}
			// No else if, as we have to hide the tipping element again.
			if !show {
				tb.tabs[index].Hide()
			}
		}
	}
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
			tb.box.PackStart(tab.EventBox, false, false, 0)
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
				tb.box.ReorderChild(tabs[i].EventBox, i+toStart)
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
				tb.box.ReorderChild(tab.EventBox, i+toStart)
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
		tb.reposition()
	})
}

// FocusTab focuses the tab at the given index.
//
// Any currently focused tab is unfocused.
func (tb *TabBar) FocusTab(i int) {
	ggtk.GlibMainContextInvoke(func() {
		if tb.focused != nil {
			tb.focused.box.SetName("")
			tb.focused.redraw()
		}
		tb.tabs[i].box.SetName("focused")
		tb.tabs[i].redraw()
		tb.focused = tb.tabs[i]
		tb.reposition()
	})
}

// popTabs removes the n tabs
func (tb *TabBar) popTabs(n int) {
	ggtk.GlibMainContextInvoke(func() {
		delSlice := tb.tabs[len(tb.tabs)-n:]
		for i, tab := range delSlice {
			tb.box.Remove(tab)
			if tab == tb.focused {
				tb.focused = nil
			}
			tab.Free()
			delSlice[i] = nil
		}
		tb.tabs = tb.tabs[:len(tb.tabs)-n]
		if len(tb.tabs) == 0 {
			for _, handle := range tb.handles {
				tb.ebox.HandlerDisconnect(handle)
			}
		}
		tb.reposition()
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
	box          *gtk.Box
	image        *gtk.Widget
	label        *gtk.Label
	parent       *TabBar
	title        string
	index        int
	loadProgress float64
	handles      []glib.SignalHandle
}

// newTabBarTab creates a new TabBarTab in a given TabBar at a given index.
func newTabBarTab(parent *TabBar, index int) (*TabBarTab, error) {
	box, err := gtk.EventBoxNew()
	if err != nil {
		return nil, err
	}
	box.SetHAlign(gtk.ALIGN_FILL)

	hbox, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 3)
	if err != nil {
		return nil, err
	}
	box.Add(hbox)

	cImage := C.gtk_image_new_from_surface(nil)
	if cImage == nil {
		return nil, errNilPtr
	}
	image := &gtk.Widget{glib.InitiallyUnowned{&glib.Object{
		glib.ToGObject(unsafe.Pointer(cImage))}}}
	image.Object.RefSink()
	runtime.SetFinalizer(image.Object, (*glib.Object).Unref)
	image.SetSizeRequest(16, 16)
	hbox.PackStart(image, false, false, 0)

	l, err := gtk.LabelNew("")
	if err != nil {
		return nil, err
	}
	l.SetHAlign(gtk.ALIGN_START)
	l.SetEllipsize(pango.ELLIPSIZE_END)
	hbox.PackStart(l, false, false, 0)
	l.OverrideFont("monospace")
	l.SetUseMarkup(true)
	t := &TabBarTab{
		box,
		hbox,
		image,
		l,
		parent,
		"",
		index,
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

// SetIcon sets the icon the the supplied pointer to a cairo_surface_t.
func (t *TabBarTab) SetIcon(to uintptr) {
	var surface *C.cairo_surface_t
	if to != 0 {
		size := C.double(t.box.GetAllocatedHeight())
		surface = C.scaleSurface(
			(*C.cairo_surface_t)(unsafe.Pointer(to)), size, size)
		defer C.cairo_surface_destroy(surface)
	}
	ggtk.GlibMainContextInvoke(func() {
		cimage := (*C.GtkImage)(unsafe.Pointer(t.image.Native()))
		C.gtk_image_set_from_surface(
			cimage,
			surface)
	})
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
	if t.loadProgress == 1.0 {
		text = fmt.Sprintf(
			t.parent.fmtString,
			t.index+1,
			html.EscapeString(title))
	} else {
		text = fmt.Sprintf(
			t.parent.fmtLoadString,
			t.index+1,
			int(t.loadProgress*100),
			html.EscapeString(title))
	}
	t.label.SetMarkup(t.parent.parent.MarkupReplacer.Replace(text))
}
