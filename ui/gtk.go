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

type invokation struct {
	f    interface{}
	args []interface{}
}

func (i *invokation) invoke() {
	// invoke does not check for type correctness. reflect itself panics if
	// types aren't correct, and this behaviour is desired.
	fRef := reflect.ValueOf(i.f)
	args := make([]reflect.Value, len(i.args))
	for i, arg := range i.args {
		args[i] = reflect.ValueOf(arg)
	}
	fRef.Call(args)
}

//export cgoInvoke
func cgoInvoke(ptr C.gpointer) C.gboolean {
	inv := (*invokation)(unsafe.Pointer(ptr))
	delete(activeInvokations, inv)
	inv.invoke()
	return 0
}

func GlibMainContextInvoke(f interface{}, args ...interface{}) {
	inv := &invokation{f, args}
	activeInvokations[inv] = inv
	C.go_invoke(C.gpointer(unsafe.Pointer(inv)))
}
