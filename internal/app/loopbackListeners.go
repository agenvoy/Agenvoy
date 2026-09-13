package app

import (
	"log/slog"
	"net"
)

func LoopbackListeners(port string) ([]net.Listener, error) {
	var listeners []net.Listener
	var firstErr error

	for _, host := range []string{"127.0.0.1", "[::1]"} {
		listener, err := net.Listen("tcp", host+":"+port)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			slog.Warn("net.Listen",
				slog.String("addr", host+":"+port),
				slog.String("error", err.Error()))
			continue
		}
		listeners = append(listeners, listener)
	}

	if len(listeners) == 0 {
		return nil, firstErr
	}
	return listeners, nil
}
