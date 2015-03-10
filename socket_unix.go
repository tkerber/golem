package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/tkerber/golem/golem"
	"github.com/tkerber/golem/xdg"
)

const socketTimeout = 50 * time.Millisecond

func socketFile(profile string) string {
	return filepath.Join(
		xdg.GetUserRuntimeDir(),
		fmt.Sprintf("golem-%s", profile))
}

func acquireSocket(
	profile string,
	socketAcquired func(net.Listener),
	socketFound func(net.Conn)) {

	// Try to acquire golem's socket.
	//
	// Step 1: Look for the socket file. If non exists, congrats, you own it.
	// Step 2: Connect to the socket.
	// Step 3: If step 2 didn't complete within some timeframe (50ms), consider
	//         the socket dead and remove it.
	// Step 4: Otherwise, maintain the connection to the socket, and send
	//         instructions.
	fpath := socketFile(profile)
	// stat it. If its a socket, ping it, if it doesn't exist, create & own it,
	// else crash and burn.
	stat, err := os.Stat(fpath)
	if os.IsNotExist(err) {
		// acquire socket
		listener, err := net.Listen("unix", fpath)
		if err != nil {
			golem.Errlog.Printf(
				"Failed to listen at socket '%s': %v", fpath, err)
			exitCode = 1
			return
		}
		socketAcquired(listener)
	} else if err != nil {
		golem.Errlog.Printf(
			"Failed to stat golem socket file '%s': %v", fpath, err)
		exitCode = 1
		return
	} else if stat.Mode()&os.ModeSocket != 0 {
		// connect
		conn, err := net.DialTimeout("unix", fpath, socketTimeout)
		if err != nil {
			// socket considered dead. Delete and re-create.
			err = os.Remove(fpath)
			if err != nil {
				golem.Errlog.Printf(
					"Failed to remove dead socket '%s': %v", fpath, err)
				exitCode = 1
				return
			}
			listener, err := net.Listen("unix", fpath)
			if err != nil {
				golem.Errlog.Printf(
					"Failed to listen at socket '%s': %v", fpath, err)
				exitCode = 1
				return
			}
			socketAcquired(listener)
		} else {
			socketFound(conn)
			conn.Close()
		}
	} else {
		golem.Errlog.Printf(
			"Expected socket file '%s' to be a socket. (%v)", fpath, err)
		exitCode = 1
		return
	}
}
