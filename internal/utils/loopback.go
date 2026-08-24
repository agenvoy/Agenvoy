package utils

import "net"

func IsLoopback(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	switch host {
	case "127.0.0.1", "::1":
		return true
	}
	return false
}
