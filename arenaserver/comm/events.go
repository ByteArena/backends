package comm

import "net"

type EventLog struct{ Value string }
type EventError struct{ Err error }
type EventWarn struct{ Err error }

type EventConnDisconnected struct {
	Err  error
	Conn net.Conn
}
