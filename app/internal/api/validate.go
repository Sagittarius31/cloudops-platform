package api

import (
	"errors"
	"net"
	"net/url"
	"strings"

	"cloudops-platform/internal/netguard"
)

// validateTarget checks user input and fails fast on obviously internal targets. This is the
// friendly first line of defence; the real enforcement happens at connection time in the checker
// (internal/netguard), which also covers redirects and DNS rebinding.
func validateTarget(name, rawURL string, expected int, allowPrivate bool) error {
	if strings.TrimSpace(name) == "" || len(name) > 100 {
		return errors.New("name is required (max 100 chars)")
	}
	if expected < 100 || expected > 599 {
		return errors.New("expected_status must be between 100 and 599")
	}
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return errors.New("url must be a valid http or https URL")
	}
	if allowPrivate {
		return nil
	}
	host := u.Hostname()
	var ips []net.IP
	if ip := net.ParseIP(host); ip != nil {
		ips = []net.IP{ip}
	} else {
		resolved, err := net.LookupIP(host)
		if err != nil {
			return errors.New("host could not be resolved")
		}
		ips = resolved
	}
	for _, ip := range ips {
		if netguard.IsBlocked(ip) {
			return errors.New("target resolves to a private or loopback address")
		}
	}
	return nil
}
