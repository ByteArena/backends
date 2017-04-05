package utils

import (
	"sync"
	"time"
)

func WaitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return true // completed normally
	case <-time.After(timeout):
		return false // timed out
	}
}

func ChanTimeout(ch chan Tickturn, timeout time.Duration) bool {
	select {
	case <-ch:
		return true // completed normally
	case <-time.After(timeout):
		return false // timed out
	}
}
