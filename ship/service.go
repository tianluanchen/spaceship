package ship

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"spaceship/pkg"
)

var (
	_ http.Handler = (*Service)(nil)
)

type ServiceOption struct {
	// if empty, no authentication
	Auth string
	// default "/", if not ends with "/" and starts with "/", then auto fix it
	URLPathPrefix string
	// default "./"
	Root string
}

type Service struct {
	authHash string
	// ends with "/" and starts with "/"
	prefix string
	root   string
	tasks  *taskSet
}

func (srv *Service) onUpload(absPath, relPath string, w http.ResponseWriter, r *http.Request) {
	defer srv.tasks.clean()
	if r.Method == http.MethodPost {
		var info UploadInfo
		deocder := json.NewDecoder(r.Body)
		if err := deocder.Decode(&info); err != nil {
			writeBadError(w, err.Error())
			return
		}
		if exist, isDir, err := checkPath(absPath); err != nil {
			writeInternalError(w, err.Error())
			return
		} else if isDir {
			writeBadError(w, fmt.Sprintf("%s is a directory, not supported", relPath))
			return
		} else if exist && !info.Overwrite {
			writeBadError(w, fmt.Sprintf("%s already exist", relPath))
			return
		}
		// check whether info is valid
		if info.TotalSize <= 0 || info.SliceSize <= 0 {
			writeBadError(w, "invalid upload info")
			return
		}
		// minimum 1KB
		if info.SliceSize < 1024*1 {
			info.SliceSize = 1024 * 1
		}
		// maximum 100-101
		if info.TotalSize/info.SliceSize > 100 {
			info.SliceSize = info.TotalSize / 100
		}
		info.Path = relPath
		bs := sha256.Sum256([]byte(absPath))
		task := newUploadTask(info, time.Hour*24, absPath+"-"+hex.EncodeToString(bs[:]))
		if srv.tasks.addTask(absPath, task, info.Overwrite) {
			writeSuccessWithJSON(w, task.info)
		} else {
			writeBadError(w, fmt.Sprintf("%s already uploading", relPath))
		}
	} else if r.Method == http.MethodPut {
		q := r.URL.Query()
		hash := strings.Trim(q.Get("hash"), " \n\t\r")
		index, err := strconv.Atoi(q.Get("index"))
		if err != nil || hash == "" || index < 0 {
			writeBadError(w, "invalid query")
			return
		}
		task := srv.tasks.getTask(absPath)
		if task == nil {
			writeBadError(w, fmt.Sprintf("not found upload task for %s", relPath))
			return
		}
		if task.info.TaskID != q.Get("taskID") {
			writeBadError(w, "task id not match, maybe your task is already stopped")
			return
		}
		if err := task.handle(index, hash, r.Body); err != nil {
			writeBadError(w, err.Error())
			return
		}
		if task.isFinished() {
			if v, err := pkg.CalculateFileSHA256(task.storagePath); err == nil {
				if v != task.info.Hash {
					writeBadError(w, "hash not match")
					return
				}
			} else {
				writeInternalError(w, err.Error())
				return
			}
			if err := os.Rename(task.storagePath, absPath); err == nil {
				writeSuccess(w, fmt.Sprintf("%s uploaded", relPath))
			} else {
				writeBadError(w, "rename file error:"+err.Error())
			}
		} else {
			writeSuccess(w, "")
		}
	}
}
func (srv *Service) onDownload(absPath, relPath string, w http.ResponseWriter, r *http.Request) {
	exist, isDir, err := checkPath(absPath)
	if err != nil {
		writeInternalError(w, err.Error())
		return
	} else if !exist {
		writeBadError(w, fmt.Sprintf("%s not exist", relPath))
		return
	} else if isDir {
		writeBadError(w, fmt.Sprintf("%s is a directory, not supported", relPath))
		return
	}
	req := r.Clone(r.Context())
	req.URL.Path = "/file"
	writeStatusHeader(w, true)
	http.ServeFile(w, req, absPath)
}
func (srv *Service) onList(absPath, relPath string, w http.ResponseWriter) {
	exist, isDir, err := checkPath(absPath)
	if err != nil {
		writeInternalError(w, err.Error())
		return
	} else if !exist {
		writeBadError(w, fmt.Sprintf("%s not exist", relPath))
		return
	} else if !isDir {
		writeBadError(w, fmt.Sprintf("%s is not a directory", relPath))
		return
	}

	if ds, err := os.ReadDir(absPath); err != nil {
		writeInternalError(w, err.Error())
	} else {
		writeStatusHeader(w, true)
		w.WriteHeader(http.StatusOK)
		for i, d := range ds {
			info, err := d.Info()
			if err != nil {
				w.Write([]byte("one error occurred: " + err.Error()))
				break
			}
			if info.IsDir() {
				continue
			}
			fo := &FileInfo{
				ModTime: info.ModTime().Unix(),
				Name:    info.Name(),
				Size:    info.Size(),
			}
			bs, err := fo.Dump()
			if err != nil {
				w.Write([]byte("one error occurred: " + err.Error()))
				break
			} else {
				bs = append(bs, '\n')
				w.Write(bs)
			}
			if f, ok := w.(http.Flusher); ok && i%10 == 9 {
				f.Flush()
			}
		}
	}
}
func (srv *Service) onDelete(absPath, relPath string, w http.ResponseWriter) {
	if absPath == srv.root {
		writeBadError(w, "root directory can't be deleted")
		return
	}
	exist, isDir, err := checkPath(absPath)
	if err != nil {
		writeInternalError(w, err.Error())
		return
	} else if !exist {
		writeBadError(w, fmt.Sprintf("%s not exist", relPath))
		return
	} else if isDir {
		writeBadError(w, fmt.Sprintf("%s is a directory, not supported", relPath))
		return
	}
	if err := os.Remove(absPath); err != nil {
		writeInternalError(w, err.Error())
	} else {
		writeSuccess(w, fmt.Sprintf("%s deleted", relPath))
	}
}
func (srv *Service) onMove(absPath, relPath, targetAbsPath, targetRelPath string, overwrite bool, w http.ResponseWriter) {
	if absPath == srv.root || targetAbsPath == srv.root {
		writeBadError(w, "root directory can't be moved")
		return
	}
	if absPath == targetAbsPath {
		writeBadError(w, "same path, invalid operation")
		return
	}
	exist, isDir, err := checkPath(absPath)
	if err != nil {
		writeInternalError(w, err.Error())
		return
	} else if !exist {
		writeBadError(w, fmt.Sprintf("%s not exist", relPath))
		return
	} else if isDir {
		writeBadError(w, fmt.Sprintf("%s is a directory, not supported", relPath))
		return
	}

	exist, isDir, err = checkPath(targetAbsPath)
	if err != nil {
		writeInternalError(w, err.Error())
		return
	} else if isDir {
		writeBadError(w, fmt.Sprintf("%s is a directory, invalid operation", targetRelPath))
		return
	} else if exist && !overwrite {
		writeBadError(w, fmt.Sprintf("%s already exist", targetRelPath))
		return
	}

	if err := os.Rename(absPath, targetAbsPath); err != nil {
		writeInternalError(w, err.Error())
	} else {
		writeSuccess(w, fmt.Sprintf("%s moved to %s", relPath, targetRelPath))
	}

}

/*
routes(if prefix == "/"):

ping: GET  /ping

list: GET	/list

upload: POST/PUT  /upload?path=<path>

delete: DELETE  /delete?path=<path>

download: GET/HEAD  /download?path=<path>
*/
func (srv *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, srv.prefix) {
		writeBadError(w, "Wrong URL path prefix")
		return
	}
	if srv.authHash != "" && srv.authHash != r.Header.Get(AuthHeader) {
		writeBadError(w, "Authentication failed")
		return
	}

	subURLPath := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, srv.prefix), "/")
	q := r.URL.Query()
	// ping
	if subURLPath == "ping" && r.Method == http.MethodGet {
		writeSuccess(w, "pong")
		return
	}
	relPath := CleanPath(q.Get("path"))
	if getDirCount(relPath) > 0 {
		writeBadError(w, "Multi-level directories are not supported")
		return
	}

	absPath := filepath.Join(srv.root, relPath)
	if filepath.Dir(absPath) != srv.root {
		writeBadError(w, fmt.Sprintf("Path %s out of bounds", relPath))
		return
	}

	if subURLPath == "list" && r.Method == http.MethodGet {
		if absPath != srv.root {
			writeBadError(w, "Multi-level directories are not supported")
			return
		}
		srv.onList(absPath, relPath, w)
		return
	}
	if subURLPath == "move" && r.Method == http.MethodPost {
		targetRelPath := CleanPath(q.Get("target"))
		targetAbsPath := filepath.Join(srv.root, targetRelPath)
		if filepath.Dir(targetAbsPath) != srv.root {
			writeBadError(w, fmt.Sprintf("Path %s out of bounds", targetRelPath))
			return
		}
		if getDirCount(targetRelPath) > 0 {
			writeBadError(w, "Multi-level directories are not supported")
			return
		}
		srv.onMove(absPath, relPath, targetAbsPath, targetRelPath, q.Get("overwrite") != "", w)
		return
	}
	if subURLPath == "delete" && r.Method == http.MethodDelete {
		srv.onDelete(absPath, relPath, w)
		return
	}
	if subURLPath == "download" && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		srv.onDownload(absPath, relPath, w, r)
		return
	}
	if subURLPath == "upload" && (r.Method == http.MethodPost || r.Method == http.MethodPut) {
		srv.onUpload(absPath, relPath, w, r)
		return
	}
	writeBadError(w, "Not supported request")
}

// if direct, then not hash
func (srv *Service) SetAuth(s string, direct ...bool) {
	if s == "" {
		srv.authHash = ""
		return
	}
	if len(direct) > 0 && direct[0] {
		srv.authHash = s
	} else {
		srv.authHash = hashAuth(s)
	}
}

func (srv *Service) GetRoot() string {
	return srv.root
}

func NewService(option ServiceOption) *Service {
	if !strings.HasPrefix(option.URLPathPrefix, "/") {
		option.URLPathPrefix = "/" + option.URLPathPrefix
	}
	if !strings.HasSuffix(option.URLPathPrefix, "/") {
		option.URLPathPrefix = option.URLPathPrefix + "/"
	}
	if option.Root == "" {
		option.Root = "."
	}
	srv := &Service{
		prefix: option.URLPathPrefix,
		root:   filepath.Clean(option.Root),
		tasks:  newTaskSet(),
	}
	srv.SetAuth(option.Auth)
	return srv
}
