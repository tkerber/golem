package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/rpc/jsonrpc"
	"os"
	"regexp"
	"runtime"
	"time"

	"github.com/conformal/gotk3/gtk"
	"github.com/mattn/go-shellwords"
	"github.com/tkerber/golem/golem"
)

// Build web extension & pdf.js
//go:generate make all
// Pack data
//go:generate go-bindata -o golem/bindata.go -pkg golem -nomemcopy -prefix data data/...
// Generate version constants
//go:generate go-version -o golem/version/version.go -pkg version

// exitCode contains the exit code that golem should exit with.
var exitCode = 0

const connectionSetupWait = 1000 * time.Millisecond

// main runs golem (yay!)
func main() {
	defer func() {
		rec := recover()
		if rec == nil {
			os.Exit(exitCode)
		}
		panic(rec)
	}()
	runtime.GOMAXPROCS(runtime.NumCPU())
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
		exitCode = 1
		return
	}
	args := flag.Args()

	acquireSocket(
		profile,
		func(l net.Listener) { socketAcquired(l, profile, args) },
		func(c net.Conn) { socketFound(c, args) })
}

// socketAcquired is called when golem obtains ownership of the socket, and
// starts up the browser. Note that the Listener is closed outwith this method.
func socketAcquired(l net.Listener, profile string, args []string) {
	gtk.Init(&args)
	g, err := golem.New(golem.NewRPCSession(l), profile)
	if err != nil {
		panic(fmt.Sprintf("Error during golem initialization: %v", err))
	}
	defer g.WebkitCleanup()

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
			golem.Errlog.Printf("Failed to open window: %v", err)
			exitCode = 1
			return
		}
	} else {
		// Open the last tab in the new window, then open all others in
		// order in a new tab.
		win, err := g.NewWindow(uris[0])
		if err != nil {
			golem.Errlog.Printf("Failed to open window: %v", err)
			exitCode = 1
			return
		}
		if len(uris) > 1 {
			_, err = win.NewTabs(uris[1:]...)
			if err != nil {
				golem.Errlog.Printf("Failed to open tabs: %v", err)
			}
		}
	}
	// This doesn't need to run in a goroutine, but as the gtk main
	// loop can be stopped and restarted in a goroutine, this makes
	// more sense.
	go gtk.Main()
	handleSignals(g)
	<-g.Quit
}

// handshake performs golems json-rpc client handshake with server:
//
// >>> json-rpc-client
// <<< ok
func handshake(c net.Conn) error {
	_, err := c.Write([]byte("json-rpc-client\n"))
	if err != nil {
		return err
	}
	reader := bufio.NewReader(c)
	line, _, err := reader.ReadLine()
	if err != nil {
		return err
	} else if string(line) != "ok" {
		return errors.New("Handshake failed.")
	}
	return nil
}

// socketFound is executed when a socket occupied by a running golem instance
// if found. It communicates with the running golem. (Note that the connection
// if closed outwith this function)
func socketFound(c net.Conn, args []string) {
	err := handshake(c)
	if err != nil {
		golem.Errlog.Printf("Failed to establish connection: %v", err)
		exitCode = 1
		return
	}
	rpc := jsonrpc.NewClient(c)
	// If there are no uris, instead create a new window.
	if len(args) == 0 {
		err := rpc.Call("RPCSession.NewWindow", nil, nil)
		if err != nil {
			golem.Errlog.Printf("Failed to open window: %v", err)
			exitCode = 1
			return
		}
	} else {
		err := rpc.Call("Golem.NewTabs", args, nil)
		if err != nil {
			golem.Errlog.Printf("Failed to open tabs: %v", err)
			exitCode = 1
			return
		}
	}
}
