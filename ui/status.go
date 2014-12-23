package ui

import "github.com/conformal/gotk3/gtk"

type StatusBar struct {
	CmdStatus      *gtk.Label
	LocationStatus *gtk.Label
	container      gtk.Container
}

func (s *StatusBar) SetLocationLabel(label string) {
	GlibMainContextInvoke(s.LocationStatus.SetLabel, label)
}

func (s *StatusBar) SetCmdLabel(label string) {
	GlibMainContextInvoke(s.CmdStatus.SetLabel, label)
}
