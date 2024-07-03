package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sync/atomic"
)

type FileWriter struct {
	f             *os.File
	writtenN      int64
	writeListener func(n int, index int, start, end, length int64)
}

func (fw *FileWriter) HookContext(ctx context.Context, index int, start, end, length int64, r io.Reader) error {
	bs := make([]byte, 1024*16)
	if length != -1 {
		r = io.LimitReader(r, end-start+1)
	}
	var count int64
	for {
		select {
		case <-ctx.Done():
			return errors.New("context done")
		default:
		}
		n, err := r.Read(bs)
		if n > 0 {
			if _, err := fw.f.WriteAt(bs[:n], start+count); err != nil {
				return err
			}
			atomic.AddInt64(&fw.writtenN, int64(n))
			if fw.writeListener != nil {
				fw.writeListener(n, index, start, end, length)
			}
			count += int64(n)
		}
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err

			}
		}
	}
	if length != -1 && count != end-start+1 {
		return fmt.Errorf("index: %d, start: %d, end: %d, read count %d not equal %d", index, start, end, count, end-start+1)
	}
	return nil
}
func (fh *FileWriter) OnWrite(cb func(n int, index int, start, end, length int64)) {
	fh.writeListener = cb
}
func (fh *FileWriter) WrittenN() int64 {
	return atomic.LoadInt64(&fh.writtenN)
}
func (fh *FileWriter) Truncate(size int64) error {
	return fh.f.Truncate(size)
}
func (fh *FileWriter) Close() error {
	return fh.f.Close()
}

func NewFileWriter(name string, perm ...fs.FileMode) (*FileWriter, error) {
	if len(perm) == 0 {
		perm = append(perm, 0666)
	}
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY, perm[0])
	if err != nil {
		return nil, err
	}
	return &FileWriter{
		f: f,
	}, nil
}
