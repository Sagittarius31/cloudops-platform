package netguard

import (
	"net"
	"testing"
)

func TestIsBlocked(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},
		{"::1", true},
		{"10.1.2.3", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"169.254.169.254", true}, // cloud metadata service
		{"fe80::1", true},
		{"fd00::1", true},
		{"0.0.0.0", true},
		{"224.0.0.1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"2606:4700:4700::1111", false},
	}
	for _, tc := range tests {
		ip := net.ParseIP(tc.ip)
		if ip == nil {
			t.Fatalf("bad test ip %q", tc.ip)
		}
		if got := IsBlocked(ip); got != tc.want {
			t.Errorf("IsBlocked(%s) = %v, want %v", tc.ip, got, tc.want)
		}
	}
}

func TestControl(t *testing.T) {
	if err := Control("tcp", "127.0.0.1:80", nil); err != ErrBlocked {
		t.Errorf("loopback should be blocked, got %v", err)
	}
	if err := Control("tcp", "8.8.8.8:443", nil); err != nil {
		t.Errorf("public address should be allowed, got %v", err)
	}
	if err := Control("tcp", "not-an-address", nil); err == nil {
		t.Error("malformed address should error")
	}
}
