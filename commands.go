package main

import (
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/tkerber/golem/webkit"
)

var commands = map[string]func(*window, *golem, []string){
	"open": cmdOpen,
	"bind": cmdBind,
	"set":  cmdSet,
}

func logInvalidArgs(args []string) {
	log.Printf("Invalid arguments recieved for command %v.", args[0])
}

func logNonGlobalCommand() {
	log.Printf("Non global command executed in a global contex.")
}

func cmdOpen(w *window, g *golem, args []string) {
	if w == nil {
		logNonGlobalCommand()
	}

	if len(args) < 2 {
		logInvalidArgs(args)
		return
	}
	uri := args[1]
	if regexp.MustCompile(`\w+:.*`).MatchString(uri) && len(args) == 2 {
		// We have a (hopefully) sensable protocol already. keep it.
	} else if regexp.MustCompile(`\S+\.\S+`).MatchString(uri) && len(args) == 2 {
		// What we have looks like a uri, but is missing the protocol.
		// We add http to it.

		// TODO any good way to have this sensibly default to https where
		// possible?
		uri = "http://" + uri
	} else {
		searchEngine := g.defaultSearchEngine
		s, ok := g.searchEngines[args[1]]
		var searchTerms []string
		if len(args) > 2 && ok {
			searchEngine = s
			searchTerms = args[2:len(args)]
		} else {
			searchTerms = args[1:len(args)]
		}
		uri = searchEngine.searchURI(searchTerms)
	}
	//log.Printf(uri)
	w.WebView.LoadURI(uri)
}

// cmdBind adds a binding, globally to golem.
func cmdBind(w *window, g *golem, args []string) {
	if len(args) != 3 {
		logInvalidArgs(args)
		return
	}
	g.bind(args[1], args[2])
}

const (
	qualifierGlobal uint = iota
	qualifierWindow
	qualifierTab
)

const (
	setOpSet uint = iota
	setOpAdd
	setOpSub
	setOpInvert
)

// cmdSet sets a setting.
//
// Set can take several arguments, each being of the form
//
// NAMESPACE:[QUALIFIER:]KEY=VALUE
//
// NAMESPACE may be one of: webkit, golem or w, g as shorthand
//
// If NAMESPACE is webkit, QUALIFIER may be one of global, window, tab
// or g, w, t as shorthand. By default, global is used.
//
// Currently the namespace golem doesn't accept any qualifiers.
//
// Depending on the type of setting, VALUE will be parsed differently:
//
// Boolean expressions must be:
//
// 1, 0, t, f, T, F, true, false, True, False, TRUE or FALSE
//
// Integer expressions must be in decimal form, or octal prefix with '0', or
// hexadecimal prefixed with '0x'.
//
// String expressions are not parsed and taken as-is.
//
// An error will be logged if parsing this fails, but execution will continue
// normally.
func cmdSet(w *window, g *golem, args []string) {
	for _, arg := range args[1:len(args)] {
		op := setOpSet
		split := strings.SplitN(arg, "=", 2)
		var valueStr string
		if len(split) != 2 {
			// If it ends in !, it is taken as a boolean value to be inverted.
			if strings.HasSuffix(split[0], "!") {
				op = setOpInvert
				split[0] = split[0][0 : len(split[0])-1]
				// It is easier if we have a "value" to pass around anyway.
				valueStr = "false"
			} else {
				log.Printf("Failed to parse set instruction: '%v'", arg)
				continue
			}
		} else {
			valueStr = split[1]
			if strings.HasSuffix(split[0], "+") {
				op = setOpAdd
				split[0] = split[0][0 : len(split[0])-1]
			} else if strings.HasSuffix(split[0], "-") {
				op = setOpSub
				split[0] = split[0][0 : len(split[0])-1]
			}
		}
		keyParts := strings.Split(split[0], ":")
		namespace := keyParts[0]
		var setFunc func(interface{})
		var valueType reflect.Type
		switch namespace {
		case "webkit", "w":
			var qualifier uint
			var key string
			switch len(keyParts) {
			case 3:
				switch keyParts[1] {
				case "global", "g":
					qualifier = qualifierGlobal
				case "window", "w":
					qualifier = qualifierWindow
				case "tab", "t":
					qualifier = qualifierTab
				default:
					log.Printf("Failed to parse set instruction: '%v'", arg)
					continue
				}
				key = keyParts[2]
			case 2:
				qualifier = qualifierGlobal
				key = keyParts[1]
			default:
				log.Printf("Failed to parse set instruction: '%v'", arg)
				continue
			}
			t, err := webkit.GetSettingsType(key)
			if err != nil {
				log.Printf(err.Error())
				continue
			}
			valueType = t
			var setLocal func(*webkit.Settings, interface{})
			switch valueType.Kind() {
			case reflect.Bool:
				switch op {
				case setOpSet:
					setLocal = func(s *webkit.Settings, val interface{}) {
						s.SetBool(key, val.(bool))
					}
				case setOpAdd:
					log.Printf("Cannot add to boolean value: %v", arg)
					continue
				case setOpSub:
					log.Printf("Cannot subtract from boolean value: %v", arg)
					continue
				case setOpInvert:
					setLocal = func(s *webkit.Settings, val interface{}) {
						s.SetBool(key, !s.GetBool(key))
					}
				}
			case reflect.String:
				switch op {
				case setOpSet:
					setLocal = func(s *webkit.Settings, val interface{}) {
						s.SetString(key, val.(string))
					}
				case setOpAdd:
					setLocal = func(s *webkit.Settings, val interface{}) {
						s.SetString(key, s.GetString(key)+val.(string))
					}
				case setOpSub:
					log.Printf("Cannot subtract from string value: %v", arg)
					continue
				case setOpInvert:
					log.Printf("Cannot invert string value: %v", arg)
					continue
				}
			case reflect.Uint:
				switch op {
				case setOpSet:
					setLocal = func(s *webkit.Settings, val interface{}) {
						s.SetUint(key, val.(uint))
					}
				case setOpAdd:
					setLocal = func(s *webkit.Settings, val interface{}) {
						s.SetUint(key, s.GetUint(key)+val.(uint))
					}
				case setOpSub:
					setLocal = func(s *webkit.Settings, val interface{}) {
						s.SetUint(key, s.GetUint(key)-val.(uint))
					}
				case setOpInvert:
					log.Printf("Cannot invert uint value: %v", arg)
					continue
				}
			}
			switch qualifier {
			case qualifierGlobal:
				setFunc = func(val interface{}) {
					for _, wv := range g.webViews {
						setLocal(wv.GetSettings(), val)
					}
					setLocal(g.defaultSettings, val)
				}
			case qualifierWindow:
				if w == nil {
					logNonGlobalCommand()
					continue
				}
				setFunc = func(val interface{}) {
					for _, wv := range w.webViews {
						setLocal(wv.GetSettings(), val)
					}
				}
			case qualifierTab:
				if w == nil {
					logNonGlobalCommand()
					continue
				}
				setFunc = func(val interface{}) {
					setLocal(w.getWebView().GetSettings(), val)
				}
			}
		case "golem", "g":
		default:
			log.Printf("Failed to parse set instruction: '%v'", arg)
			continue
		}
		// Parse value according to the type and apply.
		var value interface{}
		var err error
		switch valueType.Kind() {
		case reflect.Bool:
			value, err = strconv.ParseBool(valueStr)
		case reflect.String:
			value = valueStr
		case reflect.Uint:
			var v uint64
			v, err = strconv.ParseUint(valueStr, 0, 64)
			value = uint(v)
		}
		if err != nil {
			log.Printf(err.Error())
			continue
		}
		setFunc(value)
	}
}
