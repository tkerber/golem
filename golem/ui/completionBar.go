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
	labels      []*gtk.Label
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
		make([]*gtk.Label, 0, (SurroundingCompletions*2+1)*5),
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
		for i, label := range cb.labels {
			cb.Container.Remove(label)
			cb.labels[i] = nil
		}
		cb.labels = cb.labels[0:0]
	})
}

// update updates the display of completions with the specified context.
//
// Must be invoked from glib's main context.
func (cb *CompletionBar) update(completions []string, at int) error {
	labels := make([]*gtk.Label, 0, (SurroundingCompletions*2+1)*5)
	var retErr error
	for i, completion := range completions {
		split := strings.Split(completion, "\t")
		for j, str := range split {
			l, err := gtk.LabelNew("")
			if err != nil {
				retErr = err
				continue
			}
			cb.Container.Attach(l, j, i, 1, 1)
			labels = append(labels, l)
			l.SetUseMarkup(true)
			if i == at {
				l.SetMarkup(cb.parent.MarkupReplacer.Replace(
					fmt.Sprintf("<em>%s</em>", html.EscapeString(str))))
			} else {
				l.SetMarkup(cb.parent.MarkupReplacer.Replace(
					fmt.Sprintf("%s", html.EscapeString(str))))
			}
			l.SetHAlign(gtk.ALIGN_START)
			l.SetHExpand(true)
			l.SetEllipsize(pango.ELLIPSIZE_END)
			l.Show()
		}
	}
	cb.Clear()
	cb.labels = labels

	return retErr
}
