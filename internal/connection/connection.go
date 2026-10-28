package connection

import (
	"net"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

type Info struct {
	Host string
	Port string
	URL  string
}

var defaultPorts = map[string]string{
	"http":  "80",
	"https": "443",
	"ws":    "80",
	"wss":   "443",
}

var netcats = []string{"nc", "ncat", "netcat"}

func Parse(s string) Info {
	fields := strings.Fields(s)
	for i, field := range fields {
		if strings.Contains(field, "://") {
			if info, ok := parseURL(field); ok {
				return info
			}
			continue
		}
		if isNetcat(field) {
			if info, ok := parseNetcat(fields[i+1:]); ok {
				return info
			}
		}
	}
	if len(fields) == 1 {
		if info, ok := parseHostPort(fields[0]); ok {
			return info
		}
	}
	return Info{}
}

func parseURL(s string) (Info, bool) {
	u, err := url.Parse(s)
	if err != nil || u.Hostname() == "" {
		return Info{}, false
	}
	port := u.Port()
	if port == "" {
		port = defaultPorts[u.Scheme]
	}
	return Info{Host: u.Hostname(), Port: port, URL: s}, true
}

func parseNetcat(args []string) (Info, bool) {
	var operands []string
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			operands = append(operands, arg)
		}
	}
	if len(operands) < 2 || !validPort(operands[1]) {
		return Info{}, false
	}
	return Info{Host: operands[0], Port: operands[1]}, true
}

func parseHostPort(s string) (Info, bool) {
	host, port, err := net.SplitHostPort(s)
	if err != nil || host == "" || !validPort(port) {
		return Info{}, false
	}
	return Info{Host: host, Port: port}, true
}

func isNetcat(s string) bool {
	return slices.Contains(netcats, s)
}

func validPort(s string) bool {
	n, err := strconv.Atoi(s)
	return err == nil && n > 0 && n <= 65535
}
