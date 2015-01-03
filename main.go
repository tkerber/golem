package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"

	"github.com/conformal/gotk3/gtk"
	"github.com/guelfey/go.dbus"
	"github.com/guelfey/go.dbus/introspect"
	"github.com/mattn/go-shellwords"
	"github.com/tkerber/golem/golem"
)

// Build web extension & pdf.js
//go:generate make all
// Pack data
//go:generate go-bindata -o golem/bindata.go -pkg golem -nomemcopy -prefix data data/...

// main runs golem (yay!)
func main() {
	defer func() { os.Exit(exitCode) }()
	// Init command line flags.
	var profile string
	flag.StringVar(
		&profile,
		"p",
		"default",
		"Sets the profile to use. Each profile saves its data seperately, "+
			"and uses a seperate instance of Golem.")
	flag.Parse()
	if !regexp.MustCompile(`^[a-zA-Z]\w*$`).MatchString(profile) {
		fmt.Println("Please use a alphanumeric profile name starting with a letter.")
		os.Exit(1)
	}
	args := flag.Args()

	// Try to acquire the golem bus
	sBus, err := dbus.SessionBus()
	if err != nil {
		panic(fmt.Sprintf("Failed to acquire session bus: %v", err))
	}
	repl, err := sBus.RequestName(
		fmt.Sprintf(golem.DBusName, profile),
		dbus.NameFlagDoNotQueue)
	if err != nil {
		panic(fmt.Sprintf("Failed to ascertain status of Golem's bus name."))
	}
	switch repl {
	// If we get it, this is the new golem. Hurrah!
	case dbus.RequestNameReplyPrimaryOwner:
		gtk.Init(&args)
		g, err := golem.New(sBus, profile)
		defer g.WebkitCleanup()
		if err != nil {
			panic(fmt.Sprintf("Error during golem initialization: %v", err))
		}
		sBus.Export(
			g.CreateDBusWrapper(),
			golem.DBusPath,
			golem.DBusInterface)
		sBus.Export(
			introspect.Introspectable(golem.DBusIntrospection),
			golem.DBusPath,
			"org.freedesktop.DBus.Introspectable")
		// All arguments are taken as "open" commands for one tab each.
		// They will load in reverse order; i.e. with the last as the top
		// tab, to be consistent with golem's load order.
		uris := make([]string, len(args))
		for i, arg := range args {
			// we try to split it into parts to allow searches to be passed
			// via command line. If this fails, we ignore the error and just
			// pass the whole string instead.
			parts, err := shellwords.Parse(arg)
			if err != nil {
				parts = []string{arg}
			}
			uris[i] = g.OpenURI(parts)
		}
		if len(uris) == 0 {
			_, err := g.NewWindow("")
			if err != nil {
				os.Exit(1)
			}
		} else {
			// Open the last tab in the new window, then open all others in
			// order in a new tab.
			win, err := g.NewWindow(uris[len(uris)-1])
			if err != nil {
				os.Exit(1)
			}
			for _, uri := range uris[:len(uris)-1] {
				win.NewTab(uri)
			}
		}
		// This doesn't need to run in a goroutine, but as the gtk main
		// loop can be stopped and restarted in a goroutine, this makes
		// more sense.
		go gtk.Main()
		handleSignals(g)
		<-g.Quit
		sBus.ReleaseName(golem.DBusName)
	// If not, we attach to the existing one.
	default:
		o := sBus.Object(
			fmt.Sprintf(golem.DBusName, profile),
			golem.DBusPath)
		// If there are no uris, instead create a new window.
		if len(args) == 0 {
			o.Call(
				golem.DBusInterface+".NewWindow",
				0)
		} else {
			// Otherwise, create new tabs for each URI in order.
			// The tabs will be created in the 'default' window, i.e. the first.
			for _, arg := range args {
				o.Call(
					golem.DBusInterface+".NewTab",
					0,
					arg)
			}
		}
	}
}
