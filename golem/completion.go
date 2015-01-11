package golem

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mattn/go-shellwords"
	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/golem/states"
	"github.com/tkerber/golem/webkit"
)

var trailingWhitespaceRegex = regexp.MustCompile(`.*\s`)

// completeState starts completing a state.
func (w *Window) completeState(
	s cmd.State,
	cancel <-chan bool,
	first chan<- bool,
	compStates *[]cmd.State) {

	cancel2 := make(chan bool, 1)
	var strs []string
	update := make(chan bool, 1)
	go w.parent.complete(s, cancel2, update, compStates, &strs)
	go func() {
		updated := false
		for {
			select {
			case <-cancel:
				cancel2 <- true
				// destroy UI
				return
			case done := <-update:
				if !updated {
					first <- !done
					updated = true
				}
				// update UI
			}
		}
	}()
	// init UI
}

// complete retrieves the possible completions for a state and started them
// in a slice at the passed pointer.
//
// Complete is intended to be run with a go statement:
//	go complete(s, cancelCompletion, ptr)
//
// Sending to the cancel channel terminates execution of the function (at
// pre-set intervals). It is recommended to buffer the cancel channel and
// limit to sending one item, as it isn't guaranteed to be read.
//
// Passing nil for ptr is a fatal error.
func (g *Golem) complete(
	s cmd.State,
	cancel <-chan bool,
	update chan<- bool,
	compStates *[]cmd.State,
	compStrings *[]string) {

	switch s := s.(type) {
	case *cmd.NormalMode:
		g.completeNormalMode(s, cancel, update, compStates, compStrings)
	case *cmd.CommandLineMode:
		f := g.completeCommandLineMode(s)
		for {
			s, str, ok := f()
			if !ok {
				break
			}
			select {
			case <-cancel:
				return
			default:
				*compStates = append(*compStates, s)
				*compStrings = append(*compStrings, str)
				update <- false
			}
		}
		update <- true
	default:
		return
	}
}

// completeCommandLineMode completes a command line mode state.
func (g *Golem) completeCommandLineMode(
	s *cmd.CommandLineMode) func() (cmd.State, string, bool) {

	// Only the keys before the cursor are taken into account.
	keyStr := cmd.KeysStringSelective(s.CurrentKeys[:s.CursorPos], false)
	switch s.Substate {
	case states.CommandLineSubstateCommand:
		return g.completionWrapCommandLine(g.completeCommand(keyStr), s)
	default:
		return func() (cmd.State, string, bool) {
			return nil, "", false
		}
	}
}

// completionWrapCommandLine wraps a command string generator into a
// completion state generator.
func (g *Golem) completionWrapCommandLine(
	f func() (string, string, bool),
	s *cmd.CommandLineMode) func() (cmd.State, string, bool) {

	return func() (cmd.State, string, bool) {
		keyStr, desc, ok := f()
		if !ok {
			return nil, "", false
		}
		keys := cmd.ParseKeys(keyStr)
		return &cmd.CommandLineMode{
				s.StateIndependant,
				s.Substate,
				keys,
				len(keys),
				s.Finalizer,
			},
			desc,
			true
	}
}

// completeCommand Completes a command state.
func (g *Golem) completeCommand(command string) func() (string, string, bool) {
	parts, err := shellwords.Parse(command)
	if err != nil {
		return func() (string, string, bool) {
			return "", "", false
		}
	}
	// If we have a trailing whitespace, we add an empty part. This is so that
	// for "open " e.g. a uri will be completed and not a command.
	if trailingWhitespaceRegex.MatchString(command) {
		parts = append(parts, "")
	}
	if len(parts) == 1 {
		// Silly name. But we actually complete the "command" part of the
		// command here.
		return g.completeCommandCommand(command)
	}
	switch parts[0] {
	case "aqm", "addquickmark", "qm", "quickmark":
		// complete url from 2nd parameter onwards.
		return g.completeURI(parts, 2)
	case "o", "open",
		"t", "topen", "tabopen", "newtab",
		"bg", "bgopen", "backgroundopen",
		"w", "wopen", "winopen", "windowopen":
		// complete url from 1st parameter onwards.
		return g.completeURI(parts, 1)
	case "bind":
		// complete builtin/command from 2nd paramter onwards.
		return g.completeBinding(parts)
	case "set":
		// complete setting name from 1st parameter onwards.
		fallthrough
	case "rmqm", "removerequickmark":
		// complete quickmark
		return g.completeQuickmark(parts)
	case "q", "quit", "qall", "quitall":
		fallthrough
	default:
		return func() (string, string, bool) {
			return "", "", false
		}
	}
}

// completeOptionSet complets an option for the "set" command.
func (g *Golem) completeOptionSet(
	parts []string) func() (string, string, bool) {

	if len(parts) != 2 {
		return func() (string, string, bool) {
			return "", "", false
		}
	}
	i := -1
	return func() (string, string, bool) {
		for {
			i++
			if i >= len(webkit.SettingNames) {
				return "", "", false
			}
			setting := webkit.SettingNames[i]
			if strings.HasPrefix("w:"+setting, parts[1]) ||
				strings.HasPrefix("webkit:"+setting, parts[1]) {

				t, _ := webkit.GetSettingsType(setting)
				return parts[0] + " webkit:" + setting,
					fmt.Sprintf(
						"%s\t%v",
						setting,
						t),
					true
			}
		}
	}
}

// completeQuickmarks completes a quickmark argument.
func (g *Golem) completeQuickmark(
	parts []string) func() (string, string, bool) {

	qml := make([]string, 0, len(g.quickmarks))
	for qm := range g.quickmarks {
		qml = append(qml, qm)
	}
	i := -1
	return func() (string, string, bool) {
		for {
			i++
			if i >= len(qml) {
				return "", "", false
			} else if strings.HasPrefix(qml[i], parts[1]) {
				return qml[i],
					fmt.Sprintf("%s\t%s", qml[i], g.quickmarks[qml[i]]),
					true
			}
		}
	}
}

// completeBinding completes a binding argument.
func (g *Golem) completeBinding(
	parts []string) func() (string, string, bool) {

	opt := ""
	if len(parts) == 3 {
		opt = parts[2]
	} else {
		return func() (string, string, bool) {
			return "", "", false
		}
	}
	i := -1
	return func() (string, string, bool) {
		for {
			i++
			if i > len(commandNames)+len(builtinNames) {
				return "", "", false
			} else if i < len(builtinNames) {
				if strings.HasPrefix("builtin:"+builtinNames[i], opt) ||
					strings.HasPrefix("b:"+builtinNames[i], opt) {

					return parts[0] + "builtin:" + builtinNames[i],
						fmt.Sprintf("%s\tbuiltin", builtinNames[i]),
						true
				}
			} else {
				j := i - len(builtinNames)
				if strings.HasPrefix("command:"+commandNames[j], opt) ||
					strings.HasPrefix("cmd:"+commandNames[j], opt) ||
					strings.HasPrefix("c:"+commandNames[j], opt) {

					return parts[0] + "cmd:" + commandNames[j],
						fmt.Sprintf("%s\tcommand", commandNames[j]),
						true
				}
			}
		}
	}
}

// completeURI completes a URI argument.
func (g *Golem) completeURI(
	parts []string,
	startFrom int) func() (string, string, bool) {

	uriparts := parts[startFrom:]
	stage := 0
	qmArr := make([]string, len(g.quickmarks))
	i := 0
	for _, s := range g.quickmarks {
		qmArr[i] = s
		i++
	}
	i = -1
	return func() (string, string, bool) {
		var uri string
	outer:
		for {
			switch stage {
			// complete quickmarks
			case 0:
				i++
				if i >= len(qmArr) {
					stage++
					i = -1
					continue
				}
				for _, part := range uriparts {
					if !strings.Contains(qmArr[i], part) {
						continue outer
					}
				}
				uri = qmArr[i]
				break outer
			// complete history
			case 1:
				i++
				if i >= len(g.history) {
					stage++
					continue
				}
				for _, part := range uriparts {
					if !strings.Contains(g.history[i].uri, part) &&
						!strings.Contains(g.history[i].title, part) {

						continue outer
					}
				}
				uri = g.history[i].uri
				break outer
			// end iteration
			default:
				return "", "", false
			}
		}
		// Won't always cleanly work. But it doesn't have to.
		return strings.Join(parts[:startFrom], " ") + " " + uri, uri, true
	}
}

// stringCompleteAgainstList returns a function iterating over possible
// completions for the given string, amount the given list.
func stringCompleteAgainstList(
	str string,
	arr []string) func() (string, string, bool) {

	i := 0
	return func() (string, string, bool) {
		for i < len(arr) {
			if strings.HasPrefix(arr[i], str) {
				i++
				return arr[i-1], arr[i-1], true
			}
			i++
		}
		return "", "", false
	}
}

// completeCommandCommand completes the actual command of a command mode.
func (g *Golem) completeCommandCommand(
	cmd string) func() (string, string, bool) {

	commandNames := make([]string, len(commands))
	i := 0
	for command := range commands {
		commandNames[i] = command
		i++
	}
	return stringCompleteAgainstList(cmd, commandNames)
}

// completeNormalMode completes a normal mode state.
func (g *Golem) completeNormalMode(
	s *cmd.NormalMode,
	cancel <-chan bool,
	update chan<- bool,
	compStates *[]cmd.State,
	compStrings *[]string) {

	for b := range s.CurrentTree.IterLeaves() {
		select {
		case <-cancel:
			return
		default:
		}
		// We can't complete virtual keys.
		if _, ok := b.From[len(b.From)-1].(cmd.VirtualKey); ok {
			continue
		}
		// Get the new tree
		t := s.CurrentTree
		for _, k := range b.From {
			t = t.Subtrees[k]
		}
		var str string
		keysStr := cmd.KeysString(b.From)
		switch s.Substate {
		case states.NormalSubstateNormal:
			// TODO attach a short descriptive text/name/both to bindings.
			str = fmt.Sprintf("%s\t?????", keysStr)
		case states.NormalSubstateQuickmark,
			states.NormalSubstateQuickmarkTab,
			states.NormalSubstateQuickmarkWindow,
			states.NormalSubstateQuickmarksRapid:

			str = fmt.Sprintf("%s\t%s", keysStr, g.quickmarks[keysStr])
		}
		*compStates = append(*compStates, s.PredictState(b.From))
		*compStrings = append(*compStrings, str)
		update <- false
	}
	update <- true
}
