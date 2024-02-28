package ship

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/tianluanchen/spaceship/pkg"
)

func hashAuth(s string) string {
	return pkg.CalculateSHA256(s)
}

// if status is invaild, then read resp.Body  and return error
func checkRespReturnErr(resp *http.Response, first int, codes ...int) error {
	codes = append(codes, first)
	for _, v := range codes {
		if v == resp.StatusCode {
			return nil
		}
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Scan()
	s := strings.Trim(scanner.Text(), "\n\r\t ")
	if len(s) > 256 {
		s = s[:256]
	}
	return fmt.Errorf("unexpected status \"%s\" and front part of body is \"%s\"", resp.Status, s)
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

var contentRangeRegex = regexp.MustCompile(`^bytes (\d+)-(\d+)/(\d+)$`)

func parseContentRange(s string) (start, end, size int64, err error) {
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

func writeSuccessWithJSON(w http.ResponseWriter, v any) {
	w.WriteHeader(http.StatusOK)
	bs, _ := json.Marshal(v)
	w.Write(bs)
}
func writeSuccess(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(msg))
}

func writeBadError(w http.ResponseWriter, msg string) {
	http.Error(w, msg, http.StatusBadRequest)
}

func writeInternalError(w http.ResponseWriter, msg string) {
	http.Error(w, msg, http.StatusInternalServerError)
}
