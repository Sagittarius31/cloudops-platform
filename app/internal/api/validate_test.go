package api

import "testing"

func TestValidateTarget(t *testing.T) {
	tests := []struct {
		name         string
		tname        string
		url          string
		expected     int
		allowPrivate bool
		wantErr      bool
	}{
		{"valid public ip", "x", "https://8.8.8.8/health", 200, false, false},
		{"empty name", "", "https://8.8.8.8", 200, false, true},
		{"bad scheme", "x", "ftp://8.8.8.8", 200, false, true},
		{"no host", "x", "http://", 200, false, true},
		{"bad status", "x", "https://8.8.8.8", 42, false, true},
		{"loopback blocked", "x", "http://127.0.0.1:8080", 200, false, true},
		{"private blocked", "x", "http://10.0.0.5", 200, false, true},
		{"metadata ip blocked", "x", "http://169.254.169.254/latest", 200, false, true},
		{"private allowed in dev", "x", "http://10.0.0.5", 200, true, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTarget(tc.tname, tc.url, tc.expected, tc.allowPrivate)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
		})
	}
}
