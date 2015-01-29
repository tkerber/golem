package golem

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/mattn/go-shellwords"
	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/webkit"
)

// hasProtocolRegex matches if a "uri" has what looks like a protocol.
var hasProtocolRegex = regexp.MustCompile(`(http|https|file|golem|golem-unsafe|about):.*`)

// looksLikeURIRegex matches if (despite no protocol existing), a "uri" looks
// like a uri.
// TODO make sure this and hasProtocolRegex function correctly in almost all
// cases. (ipv6?)
var looksLikeURIRegex = regexp.MustCompile(`(\S+\.\S+|localhost)(:\d+)?(/.*)?`)

var commandNames []string

// commands maps a command name to the command's function.
var commands map[string]func(*Window, *Golem, []string)

// init initializes commands;
//
// This is to prevent a initialization loop. As, however, none of the commands
// are used during initialization, it is fine for them to reside in init,
// (which is executed after constant/variabel initialization.
func init() {
	commands = map[string]func(*Window, *Golem, []string){
		"noh":             cmdNoHLSearch,
		"nohlsearch":      cmdNoHLSearch,
		"aqm":             cmdAddQuickmark,
		"addquickmark":    cmdAddQuickmark,
		"o":               cmdOpen,
		"open":            cmdOpen,
		"t":               cmdTabOpen,
		"topen":           cmdTabOpen,
		"tabopen":         cmdTabOpen,
		"newtab":          cmdTabOpen,
		"bg":              cmdBackgroundOpen,
		"bgopen":          cmdBackgroundOpen,
		"backgroundopen":  cmdBackgroundOpen,
		"w":               cmdWindowOpen,
		"wopen":           cmdWindowOpen,
		"winopen":         cmdWindowOpen,
		"windowopen":      cmdWindowOpen,
		"newwindow":       cmdWindowOpen,
		"bind":            cmdBind,
		"set":             cmdSet,
		"rmqm":            cmdRemoveQuickmark,
		"removequickmark": cmdRemoveQuickmark,
		"q":               cmdQuit,
		"quit":            cmdQuit,
		"qall":            cmdQuitAll,
		"quitall":         cmdQuitAll,
		"qm":              cmdQuickmark,
		"quickmark":       cmdQuickmark,
	}
	commandNames = make([]string, 0, len(commands))
	for c := range commands {
		commandNames = append(commandNames, c)
	}
}

// logInvalidArgs prints a log message indicating that the arguments given
// where invalid.
func (w *Window) logInvalidArgs(args []string) {
	w.logErrorf("Invalid arguments recieved for command %v.", args[0])
}

// logNonGlobalCommand prints a log message indicating that a command should
// not have been executed in a global context (i.e. in golem's rc)
func logNonGlobalCommand() {
	(*Window)(nil).logError("Non global command executed in a global context.")
}

// cmdNoHLSearch removes all active highlighting from the page.
func cmdNoHLSearch(w *Window, g *Golem, args []string) {
	if w == nil {
		logNonGlobalCommand()
		return
	}
	w.getWebView().GetFindController().SearchFinish()
}

// cmdBackgroundOpen opens a new tab in the background.
func cmdBackgroundOpen(w *Window, g *Golem, args []string) {
	if w == nil {
		logNonGlobalCommand()
		return
	}
	uri := g.OpenURI(args[1:])
	_, err := w.NewTabs(uri)
	if err != nil {
		w.logErrorf("Failed to open new tab: %v", err)
	}
}

// cmdAddQuickmark adds a new quickmark and records it in the quickmarks file.
func cmdAddQuickmark(w *Window, g *Golem, args []string) {
	if len(args) != 3 {
		w.logInvalidArgs(args)
		return
	}
	sanitizedKeys := cmd.KeysString(cmd.ParseKeys(args[1]))
	if uri, ok := g.quickmarks[sanitizedKeys]; ok {
		b := false
		w.setState(cmd.NewYesNoConfirmMode(
			w.State,
			cmd.SubstateDefault,
			fmt.Sprintf(
				"Do you want to replace the existing keybinding with "+
					"quickmark '%s' (%s)?",
				sanitizedKeys,
				uri),
			&b,
			func(b bool) {
				if b {
					cmdRemoveQuickmark(w, g, []string{"", args[1]})
					cmdAddQuickmark(w, g, args)
				}
			}))
		return
	}
	// Add quickmark to current session
	g.quickmark(sanitizedKeys, args[2])
	if w != nil {
		go w.UpdateLocation()
	}
	// Append quickmark to quickmarks config file.
	f, err := os.OpenFile(g.files.quickmarks, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		w.logError(err.Error())
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "qm\t%s\t%s\n", sanitizedKeys, args[2])
}

// cmdRemoveQuickmark removes a quickmark from golem and (if found) from the
// quickmarks file.
func cmdRemoveQuickmark(w *Window, g *Golem, args []string) {
	if len(args) != 2 {
		w.logInvalidArgs(args)
		return
	}
	g.wMutex.Lock()
	// First we guess that a key sequence is given, and try to delete that.
	keyStr := cmd.KeysString(cmd.ParseKeys(args[1]))
	if _, ok := g.quickmarks[args[1]]; ok {
		delete(g.hasQuickmark, g.quickmarks[keyStr])
		delete(g.quickmarks, keyStr)
	} else {
		// We assume a uri is given and try to delete that.
		found := false
		for k, v := range g.quickmarks {
			if v == args[1] {
				delete(g.quickmarks, k)
				delete(g.hasQuickmark, v)
				found = true
				break
			}
		}
		if !found {
			g.wMutex.Unlock()
			w.logErrorf("Failed to delete quickmark '%s': Not found.", args[1])
			return
		}
	}
	g.wMutex.Unlock()
	if w != nil {
		go w.UpdateLocation()
	}
	// We also run through the quickmarks file and delete matching lines.
	data, err := ioutil.ReadFile(g.files.quickmarks)
	if err != nil {
		w.logErrorf("Failed to read quickmarks file.")
		return
	}
	lines := strings.Split(string(data), "\n")
	for i := 0; i < len(lines); i++ {
		parts, err := shellwords.Parse(lines[i])
		if err != nil || len(parts) != 3 {
			continue
		}
		if parts[0] != "qm" && parts[0] != "quickmark" {
			continue
		}
		if parts[1] == keyStr || parts[2] == args[1] {
			copy(lines[i:len(lines)-1], lines[i+1:])
			lines = lines[:len(lines)-1]
			i--
		}
	}
	err = ioutil.WriteFile(
		g.files.quickmarks,
		[]byte(strings.Join(lines, "\n")),
		0600)
	if err != nil {
		w.logErrorf("Failed to write to quickmarks file.")
	}
}

// cmdQuickmark adds a new quickmark to golem.
func cmdQuickmark(w *Window, g *Golem, args []string) {
	if len(args) != 3 {
		w.logInvalidArgs(args)
		return
	}
	sanitizedKeys := cmd.KeysString(cmd.ParseKeys(args[1]))
	if uri, ok := g.quickmarks[sanitizedKeys]; ok {
		b := false
		w.setState(cmd.NewYesNoConfirmMode(
			w.State,
			cmd.SubstateDefault,
			fmt.Sprintf(
				"Do you want to replace the existing keybinding with "+
					"quickmark '%s' (%s)?",
				sanitizedKeys,
				uri),
			&b,
			func(b bool) {
				if b {
					cmdRemoveQuickmark(w, g, []string{"", args[1]})
					cmdQuickmark(w, g, args)
				}
			}))
		return
	}
	g.quickmark(sanitizedKeys, args[2])
	if w != nil {
		go w.UpdateLocation()
	}
}

// cmdQuit quit closes the active window.
func cmdQuit(w *Window, g *Golem, _ []string) {
	if w == nil {
		logNonGlobalCommand()
		return
	}
	w.Close()
}

// cmdQuitAll closes all of golems windows.
func cmdQuitAll(w *Window, g *Golem, _ []string) {
	g.Close()
}

// cmdOpen opens a uri in the current tab.
//
// cmdOpen is "smart" and guesses the uri's protocol, as well as interprets
// searches entered.
//
// Searches prefixed with the name of the search engine will be run through
// that search engine.
func cmdOpen(w *Window, g *Golem, args []string) {
	if w == nil {
		logNonGlobalCommand()
		return
	}
	uri := g.OpenURI(args[1:])
	if uri == "" {
		w.logInvalidArgs(args)
		return
	}
	w.getWebView().LoadURI(uri)
}

// cmdTabOpen behaves like cmdOpen, but opens the uri in a new tab. If no
// uri is given, it opens the new tab page instead.
func cmdTabOpen(w *Window, g *Golem, args []string) {
	if w == nil {
		logNonGlobalCommand()
		return
	}
	uri := g.OpenURI(args[1:])
	_, err := w.NewTabs(uri)
	if err != nil {
		w.logErrorf("Failed to open new tab: %v", err)
		return
	}
	w.TabNext()
}

// cmdWindowOpen behaves like cmdOpen, but opens the uri in a new window. If
// no uri is given, it opens the new tab page instead.
func cmdWindowOpen(w *Window, g *Golem, args []string) {
	uri := g.OpenURI(args[1:])
	g.NewWindow(uri)
}

// OpenURI gets the uri to go to for a command of the "open" class.
func (g *Golem) OpenURI(args []string) string {
	if len(args) < 1 {
		return ""
	}
	uri := args[0]
	if hasProtocolRegex.MatchString(uri) && len(args) == 1 {
		// We have a (hopefully) sensable protocol already. keep it.
		return uri
	} else if looksLikeURIRegex.MatchString(uri) && len(args) == 1 {
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
func cmdBind(w *Window, g *Golem, args []string) {
	if len(args) != 3 {
		w.logInvalidArgs(args)
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
func cmdSet(w *Window, g *Golem, args []string) {
	for _, arg := range args[1:len(args)] {
		op, keyParts, valueStr, err := cmdSetSplitOperator(arg)
		if err != nil {
			w.logErrorf("%v: '%v'", err, arg)
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
				w.logErrorf("%v: '%v'", err, arg)
				continue
			}
		case "golem", "g":
			// TODO Not yet implemented.
			fallthrough
		default:
			w.logErrorf("Failed to parse set instruction: '%v'", arg)
			continue
		}

		operatorFunc, err :=
			cmdSetOperatorFunc(op, setFunc, getFunc, valueType)
		if err != nil {
			w.logErrorf("%v: '%v'", err, arg)
			continue
		}

		// Parse value according to the type and apply.
		value, err := cmdSetParseValueString(valueStr, valueType)
		if err != nil {
			w.logError(err.Error())
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
			return nil, fmt.Errorf("Don't know how to subtract type %v",
				valueType)
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
			return nil, fmt.Errorf("Don't know how to invert type %v",
				valueType)
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
	w *Window,
	g *Golem,
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
			iterChan <- g.DefaultSettings
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
