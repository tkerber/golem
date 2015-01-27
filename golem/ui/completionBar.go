package ui

import (
	"fmt"
	"html"
	"strings"

	"github.com/conformal/gotk3/gtk"
	"github.com/conformal/gotk3/pango"
	ggtk "github.com/tkerber/golem/gtk"
)

// SurroundingCompletions counts how many completions before and after the
// current option should be displayed.
const SurroundingCompletions = 5

// A CompletionBar is a horizontal bar for displaying the current completion
// a some context surrounding it.
type CompletionBar struct {
	Container   *gtk.Grid
	widgets     []*gtk.Widget
	completions []string
	at          int
	parent      *Window
}

// newCompletionBar creates a new CompletionBar in a Window.
func (w *Window) newCompletionBar() (*CompletionBar, error) {
	grid, err := gtk.GridNew()
	if err != nil {
		return nil, err
	}

	grid.SetHAlign(gtk.ALIGN_FILL)
	grid.SetVAlign(gtk.ALIGN_END)
	grid.SetHExpand(true)
	grid.SetVExpand(false)
	grid.SetNoShowAll(true)
	grid.SetName("completionbar")

	return &CompletionBar{
		grid,
		make([]*gtk.Widget, 0, (SurroundingCompletions*2+1)*5),
		make([]string, 0),
		0,
		w,
	}, nil
}

// UpdateCompletions updates the array of available completions.
func (cb *CompletionBar) UpdateCompletions(completions []string) {
	cb.completions = completions
	cb.Update()
}

// UpdateAt updates the current completion.
func (cb *CompletionBar) UpdateAt(at int) {
	cb.at = at
	cb.Update()
}

// Update updates the display of completions
func (cb *CompletionBar) Update() error {
	var ret error
	ggtk.GlibMainContextInvoke(func() {
		if len(cb.completions) < 2*SurroundingCompletions+1 {
			ret = cb.update(cb.completions, cb.at)
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
			ret = cb.update(cb.completions[start:end], cb.at-start)
		}
	})
	return ret
}

// Clear detaches all active completions.
func (cb *CompletionBar) Clear() {
	ggtk.GlibMainContextInvoke(func() {
		for i, w := range cb.widgets {
			w.Destroy()
			cb.widgets[i] = nil
		}
		cb.widgets = cb.widgets[0:0]
	})
}

// update updates the display of completions with the specified context.
//
// Must be invoked from glib's main context.
func (cb *CompletionBar) update(completions []string, at int) error {
	widgets := make([]*gtk.Widget, 0, (SurroundingCompletions*2+1)*5)
	var retErr error
	for i, completion := range completions {
		split := strings.Split(completion, "\t")
		for j, str := range split {
			b, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
			if err != nil {
				retErr = err
				continue
			}
			b.SetHExpand(true)
			cb.Container.Attach(b, j, i, 1, 1)
			widgets = append(widgets, &b.Widget)
			l, err := gtk.LabelNew("")
			if err != nil {
				retErr = err
				continue
			}
			b.PackStart(l, false, false, 0)
			l.SetUseMarkup(true)
			if i == at {
				l.SetMarkup(cb.parent.MarkupReplacer.Replace(
					fmt.Sprintf("<em>%s</em>", html.EscapeString(str))))
				b.SetName("active")
			} else {
				l.SetMarkup(cb.parent.MarkupReplacer.Replace(
					fmt.Sprintf("%s", html.EscapeString(str))))
			}
			l.SetMaxWidthChars(50)
			l.SetEllipsize(pango.ELLIPSIZE_MIDDLE)
			l.OverrideFont("monospace")
			b.ShowAll()
		}
	}
	cb.Clear()
	cb.widgets = widgets

	return retErr
}
