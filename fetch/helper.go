package fetch

import (
	"context"
	"errors"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var contentRangeRegex = regexp.MustCompile(`^bytes (\d+)-(\d+)/(\d+)$`)

func ParseContentRange(s string) (start, end, size int64, err error) {
	s = strings.TrimSpace(s)
	result := contentRangeRegex.FindStringSubmatch(s)
	if result == nil {
		return 0, 0, 0, errors.New("cannot parse content range")
	}
	start, err = strconv.ParseInt(result[1], 10, 64)
	if err != nil {
		return 0, 0, 0, err
	}
	end, err = strconv.ParseInt(result[2], 10, 64)
	if err != nil {
		return 0, 0, 0, err
	}
	size, err = strconv.ParseInt(result[3], 10, 64)
	if err != nil {
		return 0, 0, 0, err
	}
	if start > end {
		return 0, 0, 0, errors.New("invalid content range: start>end")
	}
	if end >= size {
		return 0, 0, 0, errors.New("invalid content range: end>=size")
	}
	return start, end, size, nil
}

// map:  hostname(* match all) => ip
func GetDialContextWithHosts(resolveHostMap map[string]string, dialer *net.Dialer) func(ctx context.Context, network, addr string) (net.Conn, error) {
	if dialer == nil {
		dialer = &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}
	}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, _ := net.SplitHostPort(addr)
		resolveIP := resolveHostMap[host]
		if resolveIP == "" {
			resolveIP = resolveHostMap["*"]
		}
		if resolveIP != "" {
			addr = net.JoinHostPort(resolveIP, port)
		}
		return dialer.DialContext(ctx, network, addr)
	}
}
