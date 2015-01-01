package golem

import (
	"runtime"
	"time"
)

// schedGc schedules garbage collection to occur in a second.
//
// The delay is to allow ample time for any refereces to be cleaned up.
func schedGc() {
	go func() {
		<-time.After(time.Second)
		runtime.GC()
	}()
}
