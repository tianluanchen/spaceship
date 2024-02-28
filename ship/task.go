package ship

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type uploadTask struct {
	info          UploadInfo
	timer         *time.Timer
	stopChan      chan struct{}
	status        *sync.Map
	finishedCount int64
	l             *sync.Mutex
	// shared
	f           *os.File
	c           int64
	storagePath string
}

func (t *uploadTask) stop() {
	if !t.isStopped() {
		t.timer.Stop()
		close(t.stopChan)
		if f := t.f; f != nil {
			f.Close()
			// if not finish but stopped, auto clear file
			if t.storagePath != "" && !t.isFinished() {
				os.Remove(t.storagePath)
				t.storagePath = ""
			}
			t.f = nil
		}
	}
}
func (t *uploadTask) isStopped() bool {
	select {
	case <-t.stopChan:
		return true
	default:
		return false
	}
}
func (t *uploadTask) isFinished() bool {
	return t.finishedCount == t.info.SliceCount
}
func (t *uploadTask) sliceIsFinished(index int) bool {
	v, ok := t.status.Load(index)
	if !ok {
		return false
	}
	return v.(bool)
}
func (t *uploadTask) finishUploadSlice(index int) {
	atomic.AddInt64(&t.finishedCount, 1)
	t.status.Store(index, true)
}

// dont close file manually,use defer release() even if error is not nil
func (t *uploadTask) getFile() (*os.File, error) {
	t.l.Lock()
	defer t.l.Unlock()
	t.c += 1
	if t.f == nil {
		f, err := os.OpenFile(t.storagePath, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return nil, err
		}
		t.f = f
	}
	return t.f, nil
}

func (t *uploadTask) release() {
	t.l.Lock()
	defer t.l.Unlock()
	t.c--
	if t.c == 0 && t.f != nil {
		t.f.Close()
		t.f = nil
	}
}

func (t *uploadTask) handle(index int, hash string, r io.Reader) error {
	if index < 0 || index >= int(t.info.SliceCount) {
		return fmt.Errorf("invalid index %d", index)
	}
	if t.sliceIsFinished(index) {
		return fmt.Errorf("slice %d already finished", index)
	}
	if t.isStopped() {
		return errors.New("task is stopped")
	}
	f, err := t.getFile()
	defer t.release()
	if err != nil {
		return err
	}

	shouldCopiedSize := t.info.SliceSize
	if int64(index) == t.info.SliceCount-1 {
		shouldCopiedSize = t.info.TotalSize - t.info.SliceSize*(t.info.SliceCount-1)
	}

	bs := make([]byte, 1024*16)
	start := int64(index) * t.info.SliceSize
	var count int64
	hasher := sha256.New()
	r = io.LimitReader(r, shouldCopiedSize)
	for {
		select {
		case <-t.stopChan:
			return errors.New("task stopped")
		default:
		}
		n, err := r.Read(bs)
		if n > 0 {
			hasher.Write(bs[:n])
			if _, err := f.WriteAt(bs[:n], start+count); err != nil {
				return err
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
	if shouldCopiedSize != count {
		return fmt.Errorf("slice %d size not match", index)
	}
	if hash != hex.EncodeToString(hasher.Sum(nil)) {
		return fmt.Errorf("slice %d hash not match", index)
	}
	t.finishUploadSlice(index)
	if t.isFinished() {
		t.stop()
	}
	return nil
}

func newUploadTask(info UploadInfo, timeout time.Duration, storagePath string) *uploadTask {
	info.SliceCount = info.TotalSize / info.SliceSize
	if info.TotalSize%info.SliceSize > 0 {
		info.SliceCount++
	}
	info.TaskID = uuid.NewString()
	task := &uploadTask{
		info:        info,
		storagePath: storagePath,
		stopChan:    make(chan struct{}),
		status:      &sync.Map{},
		l:           &sync.Mutex{},
		timer:       time.NewTimer(timeout),
	}

	go func() {
		<-task.timer.C
		task.stop()
	}()
	return task
}

type taskSet struct {
	m *sync.Map
	l *sync.Mutex
}

func newTaskSet() *taskSet {
	return &taskSet{
		m: &sync.Map{},
		l: &sync.Mutex{},
	}
}

func (t *taskSet) clean() int {
	t.l.Lock()
	defer t.l.Unlock()
	s := make([]string, 0)
	t.m.Range(func(k, v any) bool {
		p := k.(string)
		if v.(*uploadTask).isStopped() {
			s = append(s, p)
		}
		return true
	})
	for _, v := range s {
		t.m.Delete(v)
	}
	return len(s)
}

func (set *taskSet) getTask(p string) *uploadTask {
	v, ok := set.m.Load(p)
	if ok {
		return v.(*uploadTask)
	}
	return nil
}

// atomic, if failed, argument task will be stopped
func (set *taskSet) addTask(p string, task *uploadTask, overwrite bool) bool {
	set.l.Lock()
	defer set.l.Unlock()
	if t := set.getTask(p); t != nil {
		if overwrite {
			t.stop()
		} else {
			task.stop()
			return false
		}
	}
	set.m.Store(p, task)
	return true
}

// func (set *taskSet) delTask(p string) {
// 	set.l.Lock()
// 	defer set.l.Unlock()
// 	if task, ok := set.m.Load(p); ok {
// 		task.(*uploadTask).stop()
// 		set.m.Delete(p)
// 	}
// }
