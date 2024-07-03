package network

import (
	"net"
	"net/url"
	"regexp"
)

func IsIP(s string) bool {
	ip := net.ParseIP(s)
	return ip != nil
}

func IsIPv6(s string) bool {
	ip := net.ParseIP(s)
	if ip != nil && ip.To16() != nil {
		return true
	} else {
		return false
	}
}

func IsIPv4(s string) bool {
	ip := net.ParseIP(s)
	if ip != nil && ip.To4() != nil {
		return true
	} else {
		return false
	}
}

// validate if it is a URL, with the default validation of the HTTP and HTTPS protocols.
func IsURL(s string, schemes ...string) bool {
	parsedURL, err := url.Parse(s)
	if err != nil {
		return false
	}
	if len(schemes) < 1 {
		schemes = []string{"http", "https"}
	}
	for _, scheme := range schemes {
		if scheme == parsedURL.Scheme {
			return true
		}
	}
	return false
}

var domainRegexp = regexp.MustCompile(`^(?i)[a-z0-9-]+(\.[a-z0-9-]+)+\.?$`)

func IsDomain(s string) bool {
	return domainRegexp.MatchString(s)
}

func IsValidPort(port int) bool {
	if port < 1 || port > 65535 {
		return false
	}
	return true
}
