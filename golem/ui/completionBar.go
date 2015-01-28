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

// MaxWidth is the maximum number of tabs in the completions (i.e. the
// maximum table width.)
const MaxWidth = 3

// A CompletionBar is a horizontal bar for displaying the current completion
// a some context surrounding it.
type CompletionBar struct {
	Container   *gtk.Grid
	boxes       [SurroundingCompletions*2 + 1][MaxWidth]*gtk.Box
	labels      [SurroundingCompletions*2 + 1][MaxWidth]*gtk.Label
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

	// Set up the grid contents.
	var labels [SurroundingCompletions*2 + 1][MaxWidth]*gtk.Label
	var boxes [SurroundingCompletions*2 + 1][MaxWidth]*gtk.Box
	for i, row := range labels {
		for j, _ := range row {
			b, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
			if err != nil {
				return nil, err
			}
			b.SetSizeRequest(100, -1)
			b.SetHExpand(true)
			grid.Attach(b, j, i, 1, 1)
			boxes[i][j] = b
			l, err := gtk.LabelNew("")
			if err != nil {
				return nil, err
			}
			l.SetHAlign(gtk.ALIGN_START)
			l.SetMaxWidthChars(70)
			l.SetEllipsize(pango.ELLIPSIZE_MIDDLE)
			l.OverrideFont("monospace")
			l.SetUseMarkup(true)
			b.PackStart(l, false, false, 0)
			l.Show()
			labels[i][j] = l
		}
	}

	return &CompletionBar{
		grid,
		boxes,
		labels,
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
		split := strings.SplitN(completion, "\t", MaxWidth)
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
