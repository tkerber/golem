package golem

import (
	"runtime"
	"time"
)

// gcIsSched tracks whether garbage collection is already schduled.
var gcIsSched = false

// schedGc schedules garbage collection to occur in a second.
//
// The delay is to allow ample time for any refereces to be cleaned up.
func schedGc() {
	if gcIsSched {
		return
	}
	go func() {
		gcIsSched = true
		<-time.After(time.Second)
		runtime.GC()
		gcIsSched = false
	}()
}
