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

// commands maps a command name to the command's function.
var commands map[string]func(*window, *golem, []string)

// init initializes commands;
//
// This is to prevent a initialization loop. As, however, none of the commands
// are used during initialization, it is fine for them to reside in init,
// (which is executed after constant/variabel initialization.
func init() {
	commands = map[string]func(*window, *golem, []string){
		"o":          cmdOpen,
		"open":       cmdOpen,
		"t":          cmdTabOpen,
		"topen":      cmdTabOpen,
		"tabopen":    cmdTabOpen,
		"newtab":     cmdTabOpen,
		"w":          cmdWindowOpen,
		"wopen":      cmdWindowOpen,
		"winopen":    cmdWindowOpen,
		"windowopen": cmdWindowOpen,
		"newwindow":  cmdWindowOpen,
		"bind":       cmdBind,
		"set":        cmdSet,
		"q":          cmdQuit,
		"quit":       cmdQuit,
	}
}

// logInvalidArgs prints a log message indicating that the arguments given
// where invalid.
func logInvalidArgs(args []string) {
	log.Printf("Invalid arguments recieved for command %v.", args[0])
}

// logNonGlobalCommand prints a log message indicating that a command should
// not have been executed in a global context (i.e. in golem's rc)
func logNonGlobalCommand() {
	log.Printf("Non global command executed in a global contex.")
}

// cmdQuit quit closes the active window.
func cmdQuit(w *window, g *golem, _ []string) {
	if w == nil {
		logNonGlobalCommand()
		return
	}
	w.Close()
}

// cmdOpen opens a uri in the current tab.
//
// cmdOpen is "smart" and guesses the uri's protocol, as well as interprets
// searches entered.
//
// Searches prefixed with the name of the search engine will be run through
// that search engine.
func cmdOpen(w *window, g *golem, args []string) {
	if w == nil {
		logNonGlobalCommand()
		return
	}
	uri := g.openURI(args[1:])
	if uri == "" {
		logInvalidArgs(args)
		return
	}
	w.WebView.LoadURI(uri)
}

// cmdTabOpen behaves like cmdOpen, but opens the uri in a new tab. If no
// uri is given, it opens the new tab page instead.
func cmdTabOpen(w *window, g *golem, args []string) {
	if w == nil {
		logNonGlobalCommand()
		return
	}
	uri := g.openURI(args[1:])
	w.newTab(uri)
}

// cmdWindowOpen behaves like cmdOpen, but opens the uri in a new window. If
// no uri is given, it opens the new tab page instead.
func cmdWindowOpen(w *window, g *golem, args []string) {
	uri := g.openURI(args[1:])
	g.newWindow(g.defaultSettings, uri)
}

// openURI gets the uri to go to for a command of the "open" class.
func (g *golem) openURI(args []string) string {
	if len(args) < 1 {
		return ""
	}
	uri := args[0]
	if regexp.MustCompile(`\w+:.*`).MatchString(uri) && len(args) == 1 {
		// We have a (hopefully) sensable protocol already. keep it.
		return uri
	} else if regexp.MustCompile(`\S+\.\S+`).MatchString(uri) && len(args) == 1 {
		// What we have looks like a uri, but is missing the protocol.
		// We add http to it.

		// TODO any good way to have this sensibly default to https where
		// possible?
		return "http://" + uri
	} else {
		return g.searchEngines.searchURI(args)
	}
}

// cmdBind adds a binding, globally to golem.
func cmdBind(w *window, g *golem, args []string) {
	if len(args) != 3 {
		logInvalidArgs(args)
		return
	}
	g.bind(args[1], args[2])
}

// These constants describe whether a setting should be set for all of golem,
// the current window only or the current tab only respectively.
const (
	qualifierGlobal uint = iota
	qualifierWindow
	qualifierTab
)

// These constants describe the operation used to set a value.
//
// Just set it, increment it, decrement it or invert it.
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

		var setFunc func(obj interface{}, val interface{})
		var getFunc func(obj interface{}) interface{}
		var iterChan <-chan interface{}
		var valueType reflect.Type

		switch namespace {
		case "webkit", "w":
			setFunc, getFunc, iterChan, valueType, err =
				cmdSetWebkit(w, g, keyParts)
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

		operatorFunc, err := cmdSetOperatorFunc(op, setFunc, getFunc, valueType)
		if err != nil {
			log.Printf("%v: '%v'", err, arg)
			continue
		}

		// Parse value according to the type and apply.
		value, err := cmdSetParseValueString(valueStr, valueType)
		if err != nil {
			log.Printf(err.Error())
			continue
		}
		for obj := range iterChan {
			operatorFunc(obj, value)
		}
	}
}

// cmdSetOperatorFunc combines setter and getter functions for a specifies
// type with the operation to create a final "operator" function.
func cmdSetOperatorFunc(
	op uint,
	setFunc func(obj interface{}, val interface{}),
	getFunc func(obj interface{}) interface{},
	valueType reflect.Type) (func(obj interface{}, val interface{}), error) {

	switch op {
	case setOpSet:
		return setFunc, nil
	case setOpAdd:
		switch valueType.Kind() {
		case reflect.Bool:
			return nil, fmt.Errorf("Cannot add to bool value")
		case reflect.String:
			return func(obj interface{}, val interface{}) {
				setFunc(obj, getFunc(obj).(string)+val.(string))
			}, nil
		case reflect.Uint:
			return func(obj interface{}, val interface{}) {
				setFunc(obj, getFunc(obj).(uint)+val.(uint))
			}, nil
		default:
			return nil, fmt.Errorf("Don't know how to add type %v", valueType)
		}
	case setOpSub:
		switch valueType.Kind() {
		case reflect.Bool:
			return nil, fmt.Errorf("Cannot subtract from bool value")
		case reflect.String:
			return nil, fmt.Errorf("Cannot subtract from string value")
		case reflect.Uint:
			return func(obj interface{}, val interface{}) {
				setFunc(obj, getFunc(obj).(uint)-val.(uint))
			}, nil
		default:
			return nil, fmt.Errorf("Don't know how to subtract type %v", valueType)
		}
	case setOpInvert:
		switch valueType.Kind() {
		case reflect.Bool:
			return func(obj interface{}, val interface{}) {
				setFunc(obj, !getFunc(obj).(bool))
			}, nil
		case reflect.String:
			return nil, fmt.Errorf("Cannot invert string value")
		case reflect.Uint:
			return nil, fmt.Errorf("Cannot invert uint value")
		default:
			return nil, fmt.Errorf("Don't know how to invert type %v", valueType)
		}
	default:
		panic("unreachable")
	}
}

// cmdSetParseValueString parses a string representation of a value into a
// concrete value of specified type.
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

// cmdSetSplitOperator splits a set instruction into an operator, it's key
// parts, and a string representation of the value.
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

// cmdSetWebkit retrieves getter and setter functions as well as an iterator
// and the type of the value for specified key parts to access webkit settings.
func cmdSetWebkit(
	w *window,
	g *golem,
	keyParts []string) (

	func(obj interface{}, val interface{}),
	func(obj interface{}) interface{},
	<-chan interface{},
	reflect.Type,
	error) {

	qualifier, key, err := cmdSetWebkitGetKeys(keyParts)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	valueType, err := webkit.GetSettingsType(key)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// setFunc delegates to the appropriate webkit.Settings setter.
	setFunc := func(obj interface{}, val interface{}) {
		switch valueType.Kind() {
		case reflect.Bool:
			obj.(*webkit.Settings).SetBool(key, val.(bool))
		case reflect.String:
			obj.(*webkit.Settings).SetString(key, val.(string))
		case reflect.Uint:
			obj.(*webkit.Settings).SetUint(key, val.(uint))
		}
	}

	getFunc := func(obj interface{}) interface{} {
		switch valueType.Kind() {
		case reflect.Bool:
			return obj.(*webkit.Settings).GetBool(key)
		case reflect.String:
			return obj.(*webkit.Settings).GetString(key)
		case reflect.Uint:
			return obj.(*webkit.Settings).GetUint(key)
		default:
			panic("Unreachable state reached!")
		}
	}

	if qualifier != qualifierGlobal && w == nil {
		return nil,
			nil,
			nil,
			nil,
			fmt.Errorf(
				"Attempted to set non-global setting in global context.")
	}
	iterChan := make(chan interface{})
	go func() {
		switch qualifier {
		case qualifierGlobal:
			iterChan <- g.defaultSettings
			for _, wv := range g.webViews {
				iterChan <- wv.GetSettings()
			}
		case qualifierWindow:
			for _, wv := range w.webViews {
				iterChan <- wv.GetSettings()
			}
		case qualifierTab:
			iterChan <- w.getWebView().GetSettings()
		}
		close(iterChan)
	}()
	return setFunc, getFunc, iterChan, valueType, nil
}

// cmdSetWebkitGetKeys converts key parts for a webkit set operation into
// the context level of the operation and the key to set.
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
