package ship

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"spaceship/fetch"
	"spaceship/pkg"

	"golang.org/x/net/context"
)

type Client struct {
	// path ends with /
	serverURL *url.URL
	fetcher   *fetch.Fetcher
}

type ClientOption struct {
	Auth string
	// if path not ends with /, then auto append it
	ServerURL string
	fetch.FetcherOption
}

// if direct, then not hash
func (c *Client) SetAuth(s string, direct ...bool) {
	if s == "" {
		c.fetcher.Header.Del(AuthHeader)
		return
	}
	if len(direct) > 0 && direct[0] {
		c.fetcher.Header.Set(AuthHeader, s)
	} else {
		c.fetcher.Header.Set(AuthHeader, hashAuth(s))
	}
}

// return clone
func (c *Client) GetServerURL() *url.URL {
	clone := *c.serverURL
	return &clone
}

func (c *Client) GetPingURL() string {
	u := c.GetServerURL()
	u.Path += "ping"
	return u.String()
}

func (c *Client) GetListURL() string {
	u := c.GetServerURL()
	u.Path += "list"
	return u.String()
}

func (c *Client) getFileURL(f string, subURLPath ...string) string {
	u := c.GetServerURL()
	if len(subURLPath) > 0 {
		u.Path += subURLPath[0]
	}
	q := u.Query()
	q.Set("path", f)
	u.RawQuery = q.Encode()
	return u.String()
}

func (c *Client) GetDownloadFileURL(remoteFile string) string {
	return c.getFileURL(remoteFile, "download")
}

func (c *Client) GetUploadFileURL(remoteFile string, taskID ...string) string {
	u := c.GetServerURL()
	q := u.Query()
	q.Set("path", remoteFile)
	if len(taskID) > 0 {
		q.Set("taskID", taskID[0])
	}
	u.RawQuery = q.Encode()
	u.Path += "upload"
	return u.String()
}

func (c *Client) GetDeleteFileURL(remoteFile string) string {
	return c.getFileURL(remoteFile, "delete")
}

func (c *Client) GetMoveFileURL(remoteFile, newRemoteFile string, overwrite bool) string {
	u := c.GetServerURL()
	q := u.Query()
	q.Set("path", remoteFile)
	q.Set("target", newRemoteFile)
	if overwrite {
		q.Set("overwrite", "true")
	}
	u.RawQuery = q.Encode()
	u.Path += "move"
	return u.String()
}

// Determine if the serverURL is available
func (c *Client) Ping() (time.Duration, error) {
	req, err := http.NewRequest(http.MethodGet, c.GetPingURL(), nil)
	if err != nil {
		return 0, err
	}
	start := time.Now()
	resp, err := c.fetcher.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if err := checkRespReturnErr(resp); err != nil {
		return 0, err
	}
	bs := make([]byte, 4)
	io.ReadAtLeast(resp.Body, bs, len(bs))
	if string(bs) == "pong" {
		return time.Since(start), nil
	} else {
		return 0, fmt.Errorf("invalid response: %s", string(bs))
	}
}

func (c *Client) List(cb func(info *FileInfo)) error {
	req, err := http.NewRequest(http.MethodGet, c.GetListURL(), nil)
	if err != nil {
		return err
	}
	resp, err := c.fetcher.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := checkRespReturnErr(resp); err != nil {
		return err
	}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		info := &FileInfo{}
		if err := info.Load(scanner.Bytes()); err != nil {
			return err
		}
		cb(info)
	}
	return scanner.Err()
}

func (c *Client) Delete(remoteFile string) error {
	req, err := http.NewRequest(http.MethodDelete, c.GetDeleteFileURL(remoteFile), nil)
	if err != nil {
		return err
	}
	resp, err := c.fetcher.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := checkRespReturnErr(resp); err != nil {
		return err
	}
	return nil
}

func (c *Client) Move(remoteFile, newRemoteFile string, overwrite bool) error {
	req, err := http.NewRequest(http.MethodPost, c.GetMoveFileURL(remoteFile, newRemoteFile, overwrite), nil)
	if err != nil {
		return err
	}
	resp, err := c.fetcher.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := checkRespReturnErr(resp); err != nil {
		return err
	}
	return nil
}

func (c *Client) ensureExistFile(remoteFile string) error {
	req, err := http.NewRequest(http.MethodGet, c.GetDownloadFileURL(remoteFile), nil)
	if err != nil {
		return err
	}
	resp, err := c.fetcher.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkRespReturnErr(resp)
}

// Get remote file to local if localFile is empty, then use remoteFile
func (c *Client) Get(concurrency int, remoteFile string, localFile string, hook func(beforeDownload bool, supported bool, length int64, n int)) error {
	if localFile == "" {
		localFile = remoteFile
	}
	if info, err := os.Stat(localFile); err == nil {
		if info.IsDir() {
			return fmt.Errorf("%s is a directory", localFile)
		}
	}
	if err := c.ensureExistFile(remoteFile); err != nil {
		return err
	}
	fw, err := fetch.NewFileWriter(localFile)
	if err != nil {
		return err
	}
	defer fw.Close()

	fileURL := c.GetDownloadFileURL(remoteFile)
	supported, length, err := c.fetcher.Inspect(fileURL)
	if err != nil {
		return err
	}
	hook(true, supported, length, 0)
	fw.OnWrite(func(n, index int, start, end, length int64) {
		hook(false, supported, length, n)
	})
	err = c.fetcher.DownloadWithManual(fileURL, supported, length, &fetch.DownloadOption{
		Concurrency: concurrency,
		HookContext: fw.HookContext,
	})
	fw.Truncate(fw.WrittenN())
	return err
}

func (c *Client) Put(concurrency int, overwrite bool, localFile string, remoteFile string, hook func(beforeUpload bool, info UploadInfo, n int)) error {
	if concurrency < 0 {
		concurrency = 1
	}
	if remoteFile == "" {
		remoteFile = localFile
	}
	uploadInfo := &UploadInfo{
		Overwrite: overwrite,
	}
	f, err := os.Open(localFile)
	if err != nil {
		return err
	}
	defer f.Close()
	if info, err := os.Stat(localFile); err == nil {
		if info.IsDir() {
			return fmt.Errorf("%s is a directory", localFile)
		}
		uploadInfo.TotalSize = info.Size()
		uploadInfo.SliceSize = uploadInfo.TotalSize / int64(concurrency)
		uploadInfo.Hash, err = pkg.CalculateFileSHA256(localFile)
		if err != nil {
			return err
		}
	} else {
		return err
	}
	bs, err := json.Marshal(uploadInfo)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.GetUploadFileURL(remoteFile), bytes.NewReader(bs))
	if err != nil {
		return err
	}

	if resp, err := c.fetcher.Do(req); err != nil {
		return err
	} else {
		if err := checkRespReturnErr(resp); err != nil {
			return err
		}
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(uploadInfo)
		resp.Body.Close()
		if err != nil {
			return err
		}
	}
	hook(true, *uploadInfo, 0)
	// upload
	var count int64
	var fatalErr error
	var once sync.Once
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan struct{}, concurrency)
	index := 0
	abort := func(err error) { cancel(); once.Do(func() { fatalErr = err }) }
	uploadURL := c.GetUploadFileURL(remoteFile, uploadInfo.TaskID)
	for count < uploadInfo.TotalSize {
		size := uploadInfo.TotalSize - count
		if size > uploadInfo.SliceSize {
			size = uploadInfo.SliceSize
		}
		wg.Add(1)
		go func(index int, offset int64) {
			defer func() {
				wg.Done()
				<-ch
			}()
			ch <- struct{}{}
			hasher := sha256.New()
			if _, err := io.Copy(hasher, io.LimitReader(&fileOffsetReader{
				f:      f,
				offset: offset,
			}, size)); err != nil {
				abort(err)
				return
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, io.LimitReader(&fileOffsetReader{
				f:      f,
				offset: offset,
				onRead: func(n int) {
					hook(false, *uploadInfo, n)
				},
			}, size))

			if err != nil {
				abort(err)
				return
			}
			q := req.URL.Query()
			q.Set("hash", hex.EncodeToString(hasher.Sum(nil)))
			q.Set("index", strconv.Itoa(index))
			req.URL.RawQuery = q.Encode()
			resp, err := c.fetcher.Do(req)
			if err != nil {
				abort(err)
				return
			}
			defer resp.Body.Close()
			if err := checkRespReturnErr(resp); err != nil {
				abort(err)
				return
			}
		}(index, count)
		index += 1
		count += size
	}
	wg.Wait()
	return fatalErr

}

// Don't set FetcherOption.ResponsePreInspector, it will be overwritten
func NewClient(opt ClientOption) (*Client, error) {
	c := &Client{}
	if u, err := pkg.ParseURL(opt.ServerURL); err != nil {
		return nil, err
	} else {
		c.serverURL = u
		if !strings.HasSuffix(c.serverURL.Path, "/") {
			c.serverURL.Path += "/"
		}
	}
	opt.ResponsePreInspector = func(when int, resp *http.Response) error {
		return checkRespReturnErr(resp)
	}
	if fetcher, err := fetch.NewFetcher(opt.FetcherOption); err != nil {
		return nil, err
	} else {
		c.fetcher = fetcher
	}
	c.SetAuth(opt.Auth)
	return c, nil
}

type fileOffsetReader struct {
	f      *os.File
	offset int64
	onRead func(n int)
}

func (r *fileOffsetReader) Read(p []byte) (int, error) {
	n, err := r.f.ReadAt(p, r.offset)
	if n > 0 {
		r.offset += int64(n)
		if r.onRead != nil {
			r.onRead(n)
		}
	}
	return n, err
}
