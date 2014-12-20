package ui

import (
	"github.com/conformal/gotk3/gtk"
)

type StatusBar struct {
	CmdStatus      *gtk.Label
	LocationStatus *gtk.Label
}

func (s *StatusBar) SetLocationLabel(label string) {
	s.LocationStatus.SetLabel(label)
}

func (s *StatusBar) SetCmdLabel(label string) {
	s.CmdStatus.SetLabel(label)
}
