package data

import (
	"testing"

	"hawx.me/code/assert"
)

func TestVerifyProfileURL(t *testing.T) {
	good := []string{
		"https://example.com/",
		"https://example.com/username",
		"https://example.com/username/.thing",
		"https://example.com/username/th..ing",
		"https://example.com/users?id=100",
	}

	for _, u := range good {
		t.Run(u, func(t *testing.T) {
			assert.Equal(t, u, ParseProfileURL(u))
		})
	}

	fix := map[string]string{
		"https://example.com":      "https://example.com/",
		"https://ExAMPLe.CoM/tEsT": "https://example.com/tEsT",
		// This is a MAY, and I'd prefer not to do it. Since the example form uses
		// type="url" you shouldn't be allowed to enter it.
		// "example.com":              "http://example.com/",
	}

	for in, out := range fix {
		t.Run(in, func(t *testing.T) {
			assert.Equal(t, out, ParseProfileURL(in))
		})
	}

	bad := map[string]string{
		"missing scheme":                     "example.com",
		"invalid scheme":                     "mailto:user@example.com",
		"contains a single-dot path segment": "https://example.com/foo/./bar",
		"contains a double-dot path segment": "https://example.com/foo/../bar",
		"contains a fragment":                "https://example.com/#me",
		"contains a username and password":   "https://user:pass@example.com/",
		"contains a port":                    "https://example.com:8443/",
		"host is an ipv4 address":            "https://172.28.92.51/",
		"host is an ipv6 address":            "https://[2001:db8:85a3::8a2e:370:7334]/",
	}

	for n, u := range bad {
		t.Run(n+" - "+u, func(t *testing.T) {
			assert.Equal(t, "", ParseProfileURL(u))
		})
	}
}

func TestVerifyClientID(t *testing.T) {
	good := []string{
		"https://example.com/",
		"https://example.com/username",
		"https://example.com/username/.thing",
		"https://example.com/username/th..ing",
		"https://example.com/users?id=100",
		"https://example.com:8443/",
		"http://127.0.0.1:8080/",
		"http://[::1]:9000/",
	}

	for _, u := range good {
		t.Run(u, func(t *testing.T) {
			assert.Equal(t, u, ParseClientID(u))
		})
	}

	fix := map[string]string{
		"https://example.com":      "https://example.com/",
		"https://ExAMPLe.CoM/tEsT": "https://example.com/tEsT",
	}

	for in, out := range fix {
		t.Run(in, func(t *testing.T) {
			assert.Equal(t, out, ParseClientID(in))
		})
	}

	bad := map[string]string{
		"missing scheme":                      "example.com",
		"invalid scheme":                      "mailto:user@example.com",
		"contains a single-dot path segment":  "https://example.com/foo/./bar",
		"contains a double-dot path segment":  "https://example.com/foo/../bar",
		"contains a fragment":                 "https://example.com/#me",
		"contains a username and password":    "https://user:pass@example.com/",
		"host is a non loopback ipv4 address": "https://172.28.92.51/",
		"host is a non loopback ipv6 address": "https://[2001:db8:85a3::8a2e:370:7334]/",
	}

	for n, u := range bad {
		t.Run(n+" - "+u, func(t *testing.T) {
			assert.Equal(t, "", ParseClientID(u))
		})
	}
}
