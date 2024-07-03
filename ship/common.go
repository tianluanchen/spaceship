package ship

import (
	"bytes"
	"encoding/json"
)

const (
	AuthHeader    = "auth"
	StatusHeader  = "status"
	StatusSuccess = "success"
	StatusFailure = "failure"
)

type UploadInfo struct {
	// server provide (relative path to root)
	Path string
	// server provide
	TaskID string
	// server provide
	SliceCount int64

	// client provide
	TotalSize int64
	// client provide, maybe server modify
	SliceSize int64
	// client provide
	Hash string
	// client provide
	Overwrite bool
}

type FileInfo struct {
	// value of time.Time.Unix()
	ModTime int64
	Name    string
	Size    int64
}

func (info *FileInfo) Load(bs []byte) error {
	return json.Unmarshal(bs, info)
}
func (info *FileInfo) Dump() ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(info); err != nil {
		return nil, err
	}
	bs := buffer.Bytes()
	// without \n
	return bs[:len(bs)-1], nil
}
