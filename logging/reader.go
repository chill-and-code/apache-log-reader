package logging

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"time"
)

// ReaderConfig represents the configuration to start the log reader
type ReaderConfig struct {
	Directory    string
	LastNMinutes int
}

// NewReader creates a new instance of log reader
func NewReader(cfg ReaderConfig) (*Reader, error) {
	files, err := ioutil.ReadDir(cfg.Directory)
	if err != nil {
		return nil, err
	}

	filesInfo := make([]os.FileInfo, 0, len(files))
	for _, fi := range files {
		if fi.IsDir() {
			continue
		}

		filesInfo = append(filesInfo, fi)
	}
	sort.Slice(filesInfo, func(i, j int) bool {
		return filesInfo[i].ModTime().Sub(filesInfo[j].ModTime()) < 0
	})

	lr := &Reader{
		cfg:       cfg,
		filesInfo: filesInfo,
		nowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}
	return lr, nil
}

// Reader represents the application log reader type
// responsible for reading logs from a given directory
// that were written in the last N minutes
type Reader struct {
	cfg       ReaderConfig
	filesInfo []os.FileInfo
	nowFunc   func() time.Time
}

// Read reads the log files using the given LogReader configuration
// and stores it inside a local bytes buffer to be displayed later
func (r *Reader) Read(ctx context.Context, w io.Writer) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		return r.read(w)
	}
}

// if there are an infinite number of log files,
// knowing the exact log rotation period may help
// skip iterations up to the very close of the log file
func (r *Reader) read(w io.Writer) error {
	logFileIndex := -1
	for i, fi := range r.filesInfo {
		nowMinusT := r.nowFunc().Add(-time.Duration(r.cfg.LastNMinutes) * time.Minute)
		if nowMinusT.Sub(fi.ModTime()) <= 0 {
			logFileIndex = i
			break
		}
	}
	if logFileIndex == -1 {
		return nil
	}

	filePath := path.Join(r.cfg.Directory, r.filesInfo[logFileIndex].Name())
	f, err := os.Open(filePath)
	defer func() { _ = f.Close() }()
	if err != nil {
		return err
	}

	nowMinusT := r.nowFunc().Add(-time.Duration(r.cfg.LastNMinutes) * time.Minute)
	file := NewFile(f)
	offset, err := file.IndexTime(nowMinusT)
	if err != nil {
		return err
	}

	others := r.filesInfo[logFileIndex+1 : len(r.filesInfo)]
	readTheRest := func() error {
		for _, fi := range others {
			chunks := r.stream(fi)
			for c := range chunks {
				if c.err != nil {
					return c.err
				}

				_, err := fmt.Fprintln(w, c.line)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	if offset < 0 {
		if logFileIndex+1 >= len(r.filesInfo) {
			return nil
		}

		nowMinusT := r.nowFunc().Add(-time.Duration(r.cfg.LastNMinutes) * time.Minute)
		fi := r.filesInfo[logFileIndex+1]
		if nowMinusT.Sub(fi.ModTime()) > 0 {
			return nil
		}
		return readTheRest()
	}

	_, err = f.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(w)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		_, err := writer.WriteString(scanner.Text() + "\n")
		if err != nil {
			return err
		}
		err = writer.Flush()
		if err != nil {
			return err
		}
	}

	return readTheRest()
}

type chunk struct {
	line string
	err  error
}

func (r *Reader) stream(fi os.FileInfo) chan chunk {
	out := make(chan chunk)
	go func() {
		filePath := path.Join(r.cfg.Directory, fi.Name())
		file, err := os.Open(filePath)
		defer func() {
			_ = file.Close()
		}()
		if err != nil {
			out <- chunk{
				err:  err,
				line: "",
			}
			return
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			out <- chunk{
				err:  nil,
				line: scanner.Text(),
			}
		}

		close(out)
	}()
	return out
}
