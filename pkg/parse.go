package pkg

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
)

func ParseFileNameByURLPath(p string) (string, error) {
	pathSlice := strings.Split(p, "/")
	name := ""
	if len(pathSlice) > 0 && pathSlice[len(pathSlice)-1] != "" {
		name = path.Clean(pathSlice[len(pathSlice)-1])
	}
	if name == "" {
		return "", errors.New("unable to parse file name")
	}
	return name, nil
}

// Parse URL as much as possible, scheme are optional, defaults are http and https
func ParseURL(s string, scheme ...string) (*url.URL, error) {
	if len(scheme) <= 0 {
		scheme = []string{"http", "https"}
	}
	s = strings.TrimSpace(s)
	u, err := url.Parse(s)
	if err != nil {
		prevErr := err
		u, err = url.Parse(scheme[0] + "://" + s)
		if err != nil {
			return nil, prevErr
		}
	}

	if u.Scheme == "" {
		u.Scheme = scheme[0]
		u, err = url.Parse(u.String())
		if err != nil {
			return nil, err
		}
	}
	if u.Hostname() == "" {
		return nil, fmt.Errorf("empty hostname")
	}
	for s := range scheme {
		if u.Scheme == scheme[s] {
			return u, nil
		}
	}
	return nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
}
