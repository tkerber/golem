package main

import (
	"fmt"
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
		uri = g.searchEngines.searchURI(args[1:])
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
		op, keyParts, valueStr, err := cmdSetSplitOperator(arg)
		if err != nil {
			log.Printf("%v: '%v'", err, arg)
			continue
		}
		namespace := keyParts[0]

		var setFunc func(interface{})
		var valueType reflect.Type

		switch namespace {
		case "webkit", "w":
			setFunc, valueType, err = cmdSetWebkit(w, g, op, keyParts)
			if err != nil {
				log.Printf("%v: '%v'", err, arg)
				continue
			}
		case "golem", "g":
			// TODO Not yet implemented.
			fallthrough
		default:
			log.Printf("Failed to parse set instruction: '%v'", arg)
			continue
		}

		// Parse value according to the type and apply.
		value, err := cmdSetParseValueString(valueStr, valueType)
		if err != nil {
			log.Printf(err.Error())
			continue
		}
		setFunc(value)
	}
}

func cmdSetParseValueString(
	valueStr string,
	valueType reflect.Type) (interface{}, error) {

	switch valueType.Kind() {
	case reflect.Bool:
		return strconv.ParseBool(valueStr)
	case reflect.String:
		return valueStr, nil
	case reflect.Uint:
		v, err := strconv.ParseUint(valueStr, 0, 64)
		return uint(v), err
	default:
		return nil, fmt.Errorf("Cannot parse type: %v", valueType)
	}
}

func cmdSetSplitOperator(arg string) (uint, []string, string, error) {
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
			return 0, nil, "", fmt.Errorf("Failed to parse set instruction")
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

	return op, keyParts, valueStr, nil
}

func cmdSetWebkit(
	w *window,
	g *golem,
	op uint,
	keyParts []string) (func(interface{}), reflect.Type, error) {

	qualifier, key, err := cmdSetWebkitGetKeys(keyParts)
	if err != nil {
		return nil, nil, err
	}

	valueType, err := webkit.GetSettingsType(key)
	if err != nil {
		return nil, nil, err
	}

	var setLocal func(*webkit.Settings, interface{})
	switch valueType.Kind() {
	case reflect.Bool:
		setLocal, err = cmdSetWebkitLocalBool(op, key)
	case reflect.String:
		setLocal, err = cmdSetWebkitLocalString(op, key)
	case reflect.Uint:
		setLocal, err = cmdSetWebkitLocalUint(op, key)
	}
	if err != nil {
		return nil, nil, err
	}

	var setFunc func(interface{})
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
			return nil,
				nil,
				fmt.Errorf(
					"Attempted to set non-global setting in global context.")
		}
		setFunc = func(val interface{}) {
			for _, wv := range w.webViews {
				setLocal(wv.GetSettings(), val)
			}
		}
	case qualifierTab:
		if w == nil {
			return nil,
				nil,
				fmt.Errorf(
					"Attempted to set non-global setting in global context.")
		}
		setFunc = func(val interface{}) {
			setLocal(w.getWebView().GetSettings(), val)
		}
	}
	return setFunc, valueType, nil
}

func cmdSetWebkitLocalUint(
	op uint,
	key string) (func(*webkit.Settings, interface{}), error) {

	switch op {
	case setOpSet:
		return func(s *webkit.Settings, val interface{}) {
			s.SetUint(key, val.(uint))
		}, nil
	case setOpAdd:
		return func(s *webkit.Settings, val interface{}) {
			s.SetUint(key, s.GetUint(key)+val.(uint))
		}, nil
	case setOpSub:
		return func(s *webkit.Settings, val interface{}) {
			s.SetUint(key, s.GetUint(key)-val.(uint))
		}, nil
	case setOpInvert:
		return nil, fmt.Errorf("Cannot invert uint value")
	default:
		panic("unreachable")
	}
}

func cmdSetWebkitLocalString(
	op uint,
	key string) (func(*webkit.Settings, interface{}), error) {

	switch op {
	case setOpSet:
		return func(s *webkit.Settings, val interface{}) {
			s.SetString(key, val.(string))
		}, nil
	case setOpAdd:
		return func(s *webkit.Settings, val interface{}) {
			s.SetString(key, s.GetString(key)+val.(string))
		}, nil
	case setOpSub:
		return nil, fmt.Errorf("Cannot subtract from string value")
	case setOpInvert:
		return nil, fmt.Errorf("Cannot invert string value")
	default:
		panic("unreachable")
	}
}

func cmdSetWebkitLocalBool(
	op uint,
	key string) (func(*webkit.Settings, interface{}), error) {

	switch op {
	case setOpSet:
		return func(s *webkit.Settings, val interface{}) {
			s.SetBool(key, val.(bool))
		}, nil
	case setOpAdd:
		return nil, fmt.Errorf("Cannot add to boolean value")
	case setOpSub:
		return nil, fmt.Errorf("Cannot subtract from boolean value")
	case setOpInvert:
		return func(s *webkit.Settings, val interface{}) {
			s.SetBool(key, !s.GetBool(key))
		}, nil
	default:
		panic("unreachable")
	}
}

func cmdSetWebkitGetKeys(keyParts []string) (uint, string, error) {
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
			return 0, "", fmt.Errorf("Failed to parse set instruction")
		}
		key = keyParts[2]
	case 2:
		qualifier = qualifierGlobal
		key = keyParts[1]
	default:
		return 0, "", fmt.Errorf("Failed to parse set instruction")
	}
	return qualifier, key, nil
}
