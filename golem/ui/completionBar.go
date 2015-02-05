package ui

// #cgo pkg-config: gtk+-3.0
// #include <gtk/gtk.h>
import "C"
import (
	"fmt"
	"html"
	"strings"
	"unsafe"

	"github.com/conformal/gotk3/glib"
	"github.com/conformal/gotk3/gtk"
	"github.com/conformal/gotk3/pango"
	ggtk "github.com/tkerber/golem/gtk"
)

// SurroundingCompletions counts how many completions before and after the
// current option should be displayed.
const SurroundingCompletions = 5

// MaxWidth is the maximum number of tabs in the completions (i.e. the
// maximum table width.)
const MaxWidth = 3

// CompletionBarSpacing is the spacing between the columns in the completion
// bar.
const CompletionBarSpacing = 5

// A CompletionBar is a horizontal bar for displaying the current completion
// a some context surrounding it.
type CompletionBar struct {
	Container   *gtk.Grid
	boxes       [SurroundingCompletions*2 + 1][MaxWidth]*gtk.Box
	labels      [SurroundingCompletions*2 + 1][MaxWidth]*gtk.Label
	dummyLabel  *gtk.Label
	completions []string
	at          int
	parent      *Window
	columns     int

	width       int
	windowWidth int
}

// newCompletionBar creates a new CompletionBar in a Window.
func (w *Window) newCompletionBar() (*CompletionBar, error) {
	grid, err := gtk.GridNew()
	if err != nil {
		return nil, err
	}
	grid.SetColumnSpacing(CompletionBarSpacing)
	grid.SetHAlign(gtk.ALIGN_FILL)
	grid.SetVAlign(gtk.ALIGN_END)
	grid.SetHExpand(true)
	grid.SetVExpand(false)
	grid.SetNoShowAll(true)
	grid.SetName("completionbar")

	// Set up the grid contents.
	var labels [SurroundingCompletions*2 + 1][MaxWidth]*gtk.Label
	var boxes [SurroundingCompletions*2 + 1][MaxWidth]*gtk.Box
	for i, row := range labels {
		for j, _ := range row {
			b, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
			if err != nil {
				return nil, err
			}
			b.SetHExpand(true)
			boxes[i][j] = b
			l, err := gtk.LabelNew("")
			if err != nil {
				return nil, err
			}
			l.SetHAlign(gtk.ALIGN_START)
			l.SetEllipsize(pango.ELLIPSIZE_MIDDLE)
			l.OverrideFont("monospace")
			l.SetUseMarkup(true)
			b.PackStart(l, false, false, 0)
			l.Show()
			labels[i][j] = l
		}
	}
	dummyLabel, err := gtk.LabelNew("")
	if err != nil {
		return nil, err
	}
	dummyLabel.Show()

	cb := &CompletionBar{
		grid,
		boxes,
		labels,
		dummyLabel,
		make([]string, 0),
		0,
		w,
		0,
		0,
		0,
	}
	// If the window size changes, we drop our size requests temporarily, to
	// allow the grid allocation size to change. Then we recalculate our
	// size requests.
	windowHandles := make([]glib.SignalHandle, 2)
	windowHandles[0], err = w.Window.Connect("size-allocate", func() {
		if width := cb.parent.Window.GetAllocatedWidth(); width != cb.windowWidth {
			cb.windowWidth = width
			for _, row := range cb.boxes {
				for _, box := range row {
					ggtk.GlibMainContextInvoke(box.SetSizeRequest, -1, -1)
				}
			}
		}
	})
	if err != nil {
		return nil, err
	}
	// Disconnect the signals again on destroy.
	windowHandles[1], err = w.Window.Connect("destroy", func() {
		for _, h := range windowHandles {
			w.Window.HandlerDisconnect(h)
		}
	})
	if err != nil {
		return nil, err
	}
	gridHandles := make([]glib.SignalHandle, 2)
	gridHandles[0], err = grid.Connect("size-allocate", func() {
		if width := cb.Container.GetAllocatedWidth(); width != cb.width {
			cb.width = width
			cb.Resize()
		}
	})
	if err != nil {
		return nil, err
	}
	// Disconnect the signals again on destroy.
	gridHandles[1], err = grid.Connect("destroy", func() {
		for _, h := range gridHandles {
			grid.HandlerDisconnect(h)
		}
	})
	if err != nil {
		return nil, err
	}
	return cb, nil
}

// setColumns sets the amount of columns shown in the completion bar.
func (cb *CompletionBar) setColumns(n int) {
	ggtk.GlibMainContextInvoke(func() {
		for i, row := range cb.boxes {
			for col, box := range row {
				if col >= n && col < cb.columns {
					cb.Container.Remove(box)
				} else if col >= cb.columns && col < n {
					cb.Container.Attach(box, col, i, 1, 1)
				}
			}
		}
	})
	cb.columns = n
}

// UpdateCompletions updates the array of available completions.
func (cb *CompletionBar) UpdateCompletions(completions []string) {
	cb.completions = completions
	columns := 0
	for _, completion := range cb.completions {
		segments := strings.Count(completion, "\t") + 1
		if segments > columns && segments <= MaxWidth {
			columns = segments
			if segments == MaxWidth {
				break
			}
		}
	}
	ggtk.GlibMainContextInvoke(func() {
		cb.setColumns(columns)
		cb.Resize()
		cb.Update()
	})
}

// Resize recomputes the sizes of items in the completion bar.
func (cb *CompletionBar) Resize() {
	widths := make([]int, cb.columns)
	for _, completion := range cb.completions {
		split := strings.SplitN(completion, "\t", cb.columns)
		for i, str := range split {
			cb.dummyLabel.SetText(html.EscapeString(str))
			size := C.gtk_requisition_new()
			C.gtk_widget_get_preferred_size(
				(*C.GtkWidget)(unsafe.Pointer(cb.dummyLabel.Native())),
				size,
				nil)
			if int(size.width) > widths[i] {
				widths[i] = int(size.width)
			}
			C.gtk_requisition_free(size)
		}
	}
	actualWidth := cb.Container.GetAllocatedWidth()
	actualWidth -= (cb.columns - 1) * CompletionBarSpacing
	changed := true
	fixed := make([]bool, cb.columns)
	nVariable := cb.columns
	for changed {
		changed = false
		for i, width := range widths {
			if fixed[i] {
				continue
			}
			if width <= actualWidth/nVariable {
				fixed[i] = true
				nVariable--
				actualWidth -= width
				continue
			}
		}
	}
	for i := range widths {
		if !fixed[i] {
			widths[i] = actualWidth / nVariable
			// This edge case *does* occur. It could probably be eliminated,
			// but it isn't really a concern.
			if widths[i] < 0 {
				widths[i] = 0
			}
		}
	}
	for _, row := range cb.boxes {
		for i, box := range row {
			if i >= cb.columns {
				break
			}
			box.SetSizeRequest(-1, -1)
		}
	}
	for _, row := range cb.boxes {
		for i, box := range row {
			if i >= cb.columns {
				break
			}
			box.SetSizeRequest(widths[i], -1)
		}
	}
}

// UpdateAt updates the current completion.
func (cb *CompletionBar) UpdateAt(at int) {
	cb.at = at
	cb.Update()
}

// Update updates the display of completions
func (cb *CompletionBar) Update() {
	ggtk.GlibMainContextInvoke(func() {
		if len(cb.completions) < 2*SurroundingCompletions+1 {
			cb.update(cb.completions, cb.at)
		} else {
			start := cb.at - SurroundingCompletions
			end := cb.at + SurroundingCompletions
			if start < 0 {
				end -= start
				start = 0
			} else if end > len(cb.completions) {
				start -= end - len(cb.completions)
				end = len(cb.completions)
			}
			cb.update(cb.completions[start:end], cb.at-start)
		}
	})
}

// Clear detaches all active completions.
func (cb *CompletionBar) Clear() {
	ggtk.GlibMainContextInvoke(func() {
		for _, row := range cb.boxes {
			for _, box := range row {
				box.Hide()
			}
		}
		for _, row := range cb.labels {
			for _, label := range row {
				label.SetMarkup("")
			}
		}
	})
}

// update updates the display of completions with the specified context.
//
// Must be invoked from glib's main context.
func (cb *CompletionBar) update(completions []string, at int) {
	cb.Clear()
	for i, completion := range completions {
		split := strings.SplitN(completion, "\t", cb.columns)
		for j, str := range split {
			cb.boxes[i][j].Show()
			if i == at {
				cb.labels[i][j].SetMarkup(cb.parent.MarkupReplacer.Replace(
					fmt.Sprintf("<em>%s</em>", html.EscapeString(str))))
				cb.boxes[i][j].SetName("active")
			} else {
				cb.labels[i][j].SetMarkup(cb.parent.MarkupReplacer.Replace(
					fmt.Sprintf("%s", html.EscapeString(str))))
				cb.boxes[i][j].SetName("")
			}
			cb.boxes[i][j].Show()
		}
	}
}
