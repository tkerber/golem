package ui

// #cgo pkg-config: glib-2.0
// #include <glib.h>
/*

extern gboolean
cgoInvoke(gpointer inv);

static inline void
go_invoke(gpointer inv) {
	g_main_context_invoke(
		NULL,
		cgoInvoke,
		inv);
}
*/
import "C"
import (
	"reflect"
	"unsafe"
)

// An invokation inv gets entered under activeInvokations[inv] = inv. This
// prevents it from being garbage collected which it's being passed through
// C and a pointer. When it is converted back, it is deleted from this map.
var activeInvokations = make(map[*invokation]*invokation, 10)

// An invokation encompasses a function and it's arguments, and is used to
// pass around a function "call".
type invokation struct {
	f    interface{}
	args []interface{}
	rets []interface{}
	done chan bool
}

// invoke invokes the invokation.
//
// No type checks are made, and if the types do not match a runtime panic will
// be caused.
func (i *invokation) invoke() {
	fRef := reflect.ValueOf(i.f)
	args := make([]reflect.Value, len(i.args))
	for i, arg := range i.args {
		args[i] = reflect.ValueOf(arg)
	}
	outs := fRef.Call(args)
	i.rets = make([]interface{}, len(outs))
	for j, out := range outs {
		i.rets[j] = out.Interface()
	}
	i.done <- true
}

//export cgoInvoke
func cgoInvoke(ptr C.gpointer) C.gboolean {
	inv := (*invokation)(unsafe.Pointer(ptr))
	delete(activeInvokations, inv)
	inv.invoke()
	return 0
}

// GlibMainContextInvoke invokes a function with the given arguments within
// glib's main context.
//
// No type checks are made, and if the types do not match a runtime panic will
// be caused.
func GlibMainContextInvoke(f interface{}, args ...interface{}) []interface{} {
	inv := &invokation{f, args, []interface{}{}, make(chan bool, 1)}
	activeInvokations[inv] = inv
	C.go_invoke(C.gpointer(unsafe.Pointer(inv)))
	<-inv.done
	return inv.rets
}
