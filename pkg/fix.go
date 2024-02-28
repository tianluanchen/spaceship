package pkg

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

// replace \ to / and  clean path
func CleanPath(s string) string {
	return path.Clean(strings.ReplaceAll(s, `\`, `/`))
}

// Resolve HTTP and HTTPS URLs as much as possible
func FixURL(s string) (*url.URL, error) {
	s = strings.TrimSpace(s)
	u, err := url.Parse(s)
	if err != nil {
		prevErr := err
		u, err = url.Parse("http://" + s)
		if err != nil {
			return nil, prevErr
		}
	}
	if u.Scheme == "" {
		u.Scheme = "http"
		u, err = url.Parse(u.String())
		if err != nil {
			return nil, err
		}
	}
	if u.Hostname() == "" {
		return nil, fmt.Errorf("empty hostname")
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}
	return u, nil
}
