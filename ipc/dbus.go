package ipc

import (
	"github.com/guelfey/go.dbus"
)

type WebExtension struct {
	*dbus.Object
}

func (w *WebExtension) GetScrollTop() (int64, error) {
	v, err := w.GetProperty("com.github.tkerber.golem.WebExtension.ScrollTop")
	if err != nil {
		return 0, err
	}
	return v.Value().(int64), nil
}

func (w *WebExtension) GetScrollLeft() (int64, error) {
	v, err := w.GetProperty("com.github.tkerber.golem.WebExtension.ScrollLeft")
	if err != nil {
		return 0, err
	}
	return v.Value().(int64), nil
}

func (w *WebExtension) GetScrollWidth() (int64, error) {
	v, err := w.GetProperty("com.github.tkerber.golem.WebExtension.ScrollWidth")
	if err != nil {
		return 0, err
	}
	return v.Value().(int64), nil
}

func (w *WebExtension) GetScrollHeight() (int64, error) {
	v, err := w.GetProperty("com.github.tkerber.golem.WebExtension.ScrollHeight")
	if err != nil {
		return 0, err
	}
	return v.Value().(int64), nil
}

func (w *WebExtension) SetScrollTop(to int64) error {
	call := w.Call(
		"org.freedesktop.DBus.Properties.Set",
		0,
		"com.github.tkerber.golem.WebExtension",
		"ScrollTop",
		dbus.MakeVariant(to))
	return call.Err
}

func (w *WebExtension) SetScrollLeft(to int64) error {
	call := w.Call(
		"org.freedesktop.DBus.Properties.Set",
		0,
		"com.github.tkerber.golem.WebExtension",
		"ScrollLeft",
		dbus.MakeVariant(to))
	return call.Err
}
