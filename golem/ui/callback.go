package ui

// A Callback is an interface implemented by the underlying logic of the
// window, and allows the UI to invoke a limited number of methods.
//
// For most things, logic will be handled in the logical package; however
// where this would be more difficult the ui package listens for events and
// passes them through the Callback.
type Callback interface {
	TabGo(index int) error
	TabPrev()
	TabNext()
}
