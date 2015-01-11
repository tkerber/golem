package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tkerber/golem/golem"
)

// handleSignals handles os signals for golem.
func handleSignals(g *golem.Golem) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	go func() {
		<-c
		exitCode = 1
		go func() {
			time.Sleep(100 * time.Millisecond)
			panic("Timeout during controlled shutdown.")
		}()
		g.Close()
	}()
}
