// Package netguard blocks connections to internal address ranges (SSRF protection).
package netguard

import (
	"errors"
	"net"
	"syscall"
)

// ErrBlocked is returned when a connection to a non-public address is refused.
var ErrBlocked = errors.New("connection to a private, loopback or link-local address is blocked")

// IsBlocked reports whether ip is in a range a user-supplied URL must never reach:
// loopback, private (RFC 1918 / ULA), link-local (includes cloud metadata 169.254.169.254),
// multicast, or unspecified.
func IsBlocked(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified()
}

// Control is a net.Dialer.Control hook. It runs AFTER DNS resolution, on the exact IP the
// socket is about to connect to, so it also stops redirects to internal hosts and DNS
// rebinding (a name that resolves to a public IP at validation time and a private IP later).
func Control(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil || IsBlocked(ip) {
		return ErrBlocked
	}
	return nil
}
