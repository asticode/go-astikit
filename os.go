package astikit

import (
	"os"
	"syscall"
)

// SignalHandler represents a func that can handle a signal
type SignalHandler func(s os.Signal)

func isTermSignal(s os.Signal) bool {
	return s == syscall.SIGABRT || s == syscall.SIGKILL || s == syscall.SIGINT || s == syscall.SIGQUIT || s == syscall.SIGTERM
}

// TermSignalHandler returns a SignalHandler that is executed only on a term signal
func TermSignalHandler(f func()) SignalHandler {
	return func(s os.Signal) {
		if isTermSignal(s) {
			f()
		}
	}
}
