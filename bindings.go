package main

import "github.com/tkerber/golem/cmd"

type binding struct {
	from string
	to   string
}

type unboundBuiltins map[string]func(*window)

var defaultBindings = []cmd.RawBinding{
	//	cmd.RawBinding{"r", "::builtin:reload"},
	//	cmd.RawBinding{"gh", "::builtin:home"},
	//	cmd.RawBinding{"gg", "::builtin:goToTop"},
	//	cmd.RawBinding{"G", "::builtin:goToBottom"},
	//	cmd.RawBinding{"j", "::builtin:scrollDown"},
	//	cmd.RawBinding{"k", "::builtin:scrollUp"},
	//	cmd.RawBinding{"h", "::builtin:scrollLeft"},
	//	cmd.RawBinding{"l", "::builtin:scrollRight"},
	//	cmd.RawBinding{":", "::builtin:commandMode"},
	//	cmd.RawBinding{"i", "::builtin:insertMode"},
	//	cmd.RawBinding{",h", "::builtin:back"},
	//	cmd.RawBinding{",l", "::builtin:forward"},
	//	cmd.RawBinding{"o", "::builtin:open"},
	cmd.RawBinding{"r", "::builtin:nop"},
	cmd.RawBinding{"gh", "::builtin:nop"},
	cmd.RawBinding{"gg", "::builtin:nop"},
	cmd.RawBinding{"G", "::builtin:nop"},
	cmd.RawBinding{"j", "::builtin:nop"},
	cmd.RawBinding{"k", "::builtin:nop"},
	cmd.RawBinding{"h", "::builtin:nop"},
	cmd.RawBinding{"l", "::builtin:nop"},
	cmd.RawBinding{":", "::builtin:nop"},
	cmd.RawBinding{"i", "::builtin:insertMode"},
	cmd.RawBinding{",h", "::builtin:nop"},
	cmd.RawBinding{",l", "::builtin:nop"},
	cmd.RawBinding{"o", "::builtin:nop"},
}

func builtinsFor(w *window) cmd.Builtins {
	return cmd.Builtins{
		"nop":        w.nop,
		"insertMode": func() { w.State = cmd.NewInsertMode(w.State) },
		//		"reload":      window.reload,
		//		"home":        window.home,
		//		"goToTop":     window.goToTop,
		//		"goToBottom":  window.goToBottom,
		//		"scrollUp":    window.scrollUp,
		//		"scrollDown":  window.scrollDown,
		//		"scrollLeft":  window.scrollLeft,
		//		"scrollRight": window.scrollRight,
		//		"commandMode": window.commandMode,
		//		"back":        window.back,
		//		"forward":     window.forward,
		//		"open":        window.open,
	}
}
