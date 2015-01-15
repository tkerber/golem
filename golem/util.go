package golem

import (
	"reflect"
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

// sliceEquals checks if two slices are equal, by doing a shallow comparison.
func sliceEquals(s1, s2 interface{}) bool {
	v1 := reflect.ValueOf(s1)
	v2 := reflect.ValueOf(s2)
	if v1.Len() != v2.Len() {
		return false
	}
	for i := 0; i < v1.Len(); i++ {
		if v1.Index(i).Interface() != v2.Index(i).Interface() {
			return false
		}
	}
	return true
}
