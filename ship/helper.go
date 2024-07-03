package ship

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"spaceship/pkg"
)

func hashAuth(s string) string {
	return pkg.CalculateSHA256(s)
}

// if status is invaild, then read resp.Body  and return error
func checkRespReturnErr(resp *http.Response) error {
	status := resp.Header.Get(StatusHeader)
	if status == StatusSuccess {
		return nil
	} else if status == StatusFailure {
		bs, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return errors.New(strings.Trim(string(bs), "\r\n\t "))
	}
	bs := make([]byte, 256)
	io.ReadAtLeast(resp.Body, bs, len(bs))
	return fmt.Errorf("can't get valid status header, status code is %s and front part of body is \"%s\"", resp.Status, string(bs))
}

// checkPath return isDir, exist, err
func checkPath(p string) (exist, isDir bool, err error) {
	info, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return false, false, nil
		}
		return false, false, err
	}
	return true, info.IsDir(), nil
}

func getDirCount(p string) int {
	count := 0
	p = filepath.Clean(p)
	for {
		parent := filepath.Dir(p)
		if parent == "." && p != ".." || parent == p {
			break
		}
		p = parent
		count++
	}
	return count
}

// replace \ to / and  clean path
func CleanPath(s string) string {
	return path.Clean(strings.ReplaceAll(s, `\`, `/`))
}

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

func writeStatusHeader(w http.ResponseWriter, success bool) {
	if success {
		w.Header().Set(StatusHeader, StatusSuccess)
	} else {
		w.Header().Set(StatusHeader, StatusFailure)
	}
}

func writeSuccessWithJSON(w http.ResponseWriter, v any) {
	writeStatusHeader(w, true)
	w.WriteHeader(http.StatusOK)
	bs, _ := json.Marshal(v)
	w.Write(bs)
}

func writeSuccess(w http.ResponseWriter, msg string) {
	writeStatusHeader(w, true)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(msg))
}

func writeBadError(w http.ResponseWriter, msg string) {
	writeStatusHeader(w, false)
	http.Error(w, msg, http.StatusBadRequest)
}

func writeInternalError(w http.ResponseWriter, msg string) {
	writeStatusHeader(w, false)
	http.Error(w, msg, http.StatusInternalServerError)
}
