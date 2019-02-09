package data

import (
	"net"
	"net/url"
	"strings"
)

func ParseProfileURL(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return ""
	}

	if parsed.Path == "" {
		parsed.Path = "/"
	}
	parsed.Host = strings.ToLower(parsed.Host)

	if (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Fragment != "" ||
		parsed.User != nil ||
		strings.Contains(parsed.Path, "/./") ||
		strings.Contains(parsed.Path, "/../") ||
		parsed.Port() != "" ||
		net.ParseIP(parsed.Hostname()) != nil {
		return ""
	}

	return parsed.String()
}

func ParseClientID(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return ""
	}

	if parsed.Path == "" {
		parsed.Path = "/"
	}
	parsed.Host = strings.ToLower(parsed.Host)

	ok := (parsed.Scheme == "http" || parsed.Scheme == "https") &&
		parsed.Fragment == "" &&
		parsed.User == nil &&
		!strings.Contains(parsed.Path, "/./") &&
		!strings.Contains(parsed.Path, "/../")
	if !ok {
		return ""
	}

	ip := net.ParseIP(parsed.Hostname())
	if ip != nil && !ip.IsLoopback() {
		return ""
	}

	return parsed.String()
}
