package fetch

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"runtime"
	"strconv"
	"sync"
)

const (
	WhenInspect = iota + 1
	WhenDownload
)

// Concurrent HTTP downloader
type Fetcher struct {
	Header http.Header
	client *http.Client
	// inspect response, use WhenInspect or WhenDownload to check when
	responsePreInspector func(when int, resp *http.Response) error
}

func (fetcher *Fetcher) inspectWithHead(url string) (supported bool, length int64, err error) {
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return
	}
	resp, err := fetcher.Do(req)
	if err != nil {
		return
	}
	err = fetcher.responsePreInspector(WhenInspect, resp)
	resp.Body.Close()
	if err != nil {
		return
	}
	if resp.Header.Get("Accept-Ranges") == "bytes" {
		supported = true
	}
	contentLength := resp.Header.Get("Content-Length")
	if contentLength == "" {
		length = -1
	} else {
		length, err = strconv.ParseInt(contentLength, 10, 64)
		if err != nil {
			return
		}
	}
	if supported && length < 0 {
		err = errors.New("supported but length < 0")
	}
	return
}

func (fetcher *Fetcher) inspectWithGet(url string) (supported bool, length int64, err error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Range", "bytes=0-0")
	resp, err := fetcher.Do(req)
	if err != nil {
		return
	}
	err = fetcher.responsePreInspector(WhenInspect, resp)
	resp.Body.Close()
	if err != nil {
		return
	}
	contentRange := resp.Header.Get("Content-Range")
	if contentRange == "" {
		supported = false
		length = -1
		return
	}
	supported = true
	start, end, length, err := ParseContentRange(contentRange)
	if start != 0 || end != 0 {
		err = errors.New("content range start or end is not 0")
	}
	return
}

// Determine if a slice download is supported
func (fetcher *Fetcher) Inspect(url string) (supported bool, length int64, err error) {
	supported, length, err = fetcher.inspectWithHead(url)
	if err != nil || !supported {
		supported, length, err = fetcher.inspectWithGet(url)
	}
	return
}

type DownloadOption struct {
	Context     context.Context
	Concurrency int
	// try times, default 3
	Try int
	// length is total body length, if length is -1, it is unknown
	HookContext func(ctx context.Context, index int, start, end, length int64, r io.Reader) error
}

// Download with specified inspect result
func (fetcher *Fetcher) DownloadWithManual(url string, supported bool, length int64, option *DownloadOption) (fatalErr error) {
	if option == nil {
		option = &DownloadOption{}
	}
	if option.Concurrency <= 0 {
		option.Concurrency = 8
	}
	if option.HookContext == nil {
		option.HookContext = func(ctx context.Context, index int, start, end, length int64, r io.Reader) error { return nil }
	}
	if option.Try <= 0 {
		option.Try = 3
	}
	if option.Context == nil {
		option.Context = context.Background()
	}
	var once sync.Once
	ctx, cancel := context.WithCancel(option.Context)
	defer cancel()
	if length <= 0 {
		length = -1
	}
	if !supported || length == -1 {
		option.Concurrency = 1
	}

	size := length / int64(option.Concurrency)
	// minimum
	if size < 1024 {
		size = 1024
	}
	var count int64
	var index int
	var wg sync.WaitGroup
	ch := make(chan struct{}, option.Concurrency)
	for count < length || length == -1 {
		var start, end int64
		if length != -1 || !supported {
			left := length - count
			if left > size {
				start = count
				count += size
				end = count - 1
			} else {
				start = count
				count += left
				end = count - 1
			}
		}
		wg.Add(1)
		go func(index int) {
			defer func() {
				wg.Done()
				<-ch
			}()
			ch <- struct{}{}
			for j := 0; j < option.Try; j++ {
				req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
				req.Header = fetcher.Header.Clone()
				if length != -1 && supported {
					req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
				}
				resp, err := fetcher.client.Do(req)
				if err == nil {
					err = fetcher.responsePreInspector(WhenDownload, resp)
				}
				if err != nil {
					if j == option.Try-1 {
						once.Do(func() {
							fatalErr = err
						})
						cancel()
						return
					}
					continue
				}
				if err := option.HookContext(ctx, index, start, end, length, resp.Body); err != nil {
					once.Do(func() {
						fatalErr = err
					})
					cancel()
				}
				resp.Body.Close()
				break
			}
		}(index)
		index++
		if length == -1 {
			break
		}
	}
	wg.Wait()
	return
}

// Download and auto inspect
func (fetcher *Fetcher) Download(url string, option *DownloadOption) error {
	supported, length, err := fetcher.Inspect(url)
	if err != nil {
		return err
	}
	return fetcher.DownloadWithManual(url, supported, length, option)
}

// send request via Fetcher.client, Fetcher.Header will be merged
func (fetcher *Fetcher) Do(req *http.Request) (*http.Response, error) {
	if req.Header == nil {
		req.Header = fetcher.Header.Clone()
	} else {
		for k, v := range fetcher.Header {
			req.Header.Del(k)
			for _, vv := range v {
				req.Header.Add(k, vv)
			}
		}
	}
	return fetcher.client.Do(req)
}

type FetcherOption struct {
	InsecureSkipVerify bool
	DisallowRedirects  bool
	DisableHTTP2       bool
	ProxyURL           string
	// * match any host
	ResolveHostMap       map[string]string
	RootCAs              *x509.CertPool
	ResponsePreInspector func(when int, resp *http.Response) error
}

func NewFetcher(option FetcherOption) (*Fetcher, error) {
	responsePreInspector := option.ResponsePreInspector
	if responsePreInspector == nil {
		responsePreInspector = func(when int, resp *http.Response) error { return nil }
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	// unlimit
	transport.MaxConnsPerHost = 0
	transport.MaxIdleConns = 0
	transport.MaxIdleConnsPerHost = 10000
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: option.InsecureSkipVerify,
		RootCAs:            option.RootCAs,
	}

	if option.DisableHTTP2 {
		// https://pkg.go.dev/net/http#hdr-HTTP_2
		transport.TLSNextProto = make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)
	}

	if option.ResolveHostMap != nil {
		transport.DialContext = GetDialContextWithHosts(option.ResolveHostMap, nil)
	}

	if option.ProxyURL != "" {
		if u, err := url.Parse(option.ProxyURL); err == nil {
			transport.Proxy = http.ProxyURL(u)
		} else {
			return nil, err
		}
	}

	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Transport: transport,
		Jar:       jar,
	}
	if option.DisallowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	ua := "Mozilla/5.0 (X11; Linux i686; rv:122.0) Gecko/20100101 Firefox/122.0"
	if runtime.GOOS == "windows" {
		ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:122.0) Gecko/20100101 Firefox/122.0"
	} else if runtime.GOOS == "darwin" {
		ua = "Mozilla/5.0 (Macintosh; Intel Mac OS X 14.3; rv:122.0) Gecko/20100101 Firefox/122.0"
	}
	return &Fetcher{
		responsePreInspector: func(when int, resp *http.Response) error {
			err := responsePreInspector(when, resp)
			if err != nil {
				defer resp.Body.Close()
			}
			return err
		},
		Header: http.Header{
			"User-Agent": []string{
				ua,
			},
		},
		client: client,
	}, nil
}
