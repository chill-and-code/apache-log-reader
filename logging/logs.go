package logging

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"time"
)

// LogsConfig represents the configuration Logs.
type LogsConfig struct {
	Directory    string
	LastNMinutes int
}

// NewLogs creates a new instance of Logs containing all the info
// about the log files to look for within a given time range.
func NewLogs(cfg LogsConfig) (*Logs, error) {
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
	// make sure to sort all the log files by the modified time
	// instead of relying on alphanumerical sorting
	sort.Slice(filesInfo, func(i, j int) bool {
		return filesInfo[i].ModTime().Sub(filesInfo[j].ModTime()) < 0
	})

	logs := &Logs{
		cfg:       cfg,
		filesInfo: filesInfo,
		nowMinusT: func() time.Time {
			return time.Now().UTC().Add(-time.Duration(cfg.LastNMinutes) * time.Minute)
		},
	}
	return logs, nil
}

// Logs represents the application Logs type
// containing information about the logs files from a given directory
// that were written in the last N minutes.
type Logs struct {
	cfg       LogsConfig
	filesInfo []os.FileInfo
	nowMinusT func() time.Time
}

// Print reads the log files using the given Logs configuration
// and streams them to a given writer.
func (logs *Logs) Print(w io.Writer) error {
	idx := logs.index()
	if idx == -1 {
		return nil
	}

	file, err := os.Open(path.Join(logs.cfg.Directory, logs.filesInfo[idx].Name()))
	if err != nil {
		return err
	}

	offset, err := NewFile(file).IndexTime(logs.nowMinusT())
	if err != nil {
		return err
	}

	if offset >= 0 {
		err = logs.streamFile(file, offset, w)
		if err != nil {
			return err
		}
	}

	// means we're reading the last file which has no fresh logs
	// so there are no other files left to stream => return.
	if idx+1 >= len(logs.filesInfo) || logs.nowMinusT().Sub(logs.filesInfo[idx+1].ModTime()) > 0 {
		return nil
	}

	rest := logs.filesInfo[idx+1 : len(logs.filesInfo)]
	return logs.streamFiles(rest, w)
}

// index returns the index (offset) of the first file that contains logs
// that have happened within the last N minutes or -1 if no file contains any fresh logs.
func (logs *Logs) index() int {
	idx := -1
	for i, fi := range logs.filesInfo {
		if logs.nowMinusT().Sub(fi.ModTime()) <= 0 {
			idx = i
			break
		}
	}

	return idx
}

// streamFiles reads from a given list of files and writes to a given writer using a given seek offset.
// Because we need to preserve the order of the logs, and we want to also immediately stream to
// a given writer, we cannot use go routines. In a different scenario where order is not important
// that can of course be very useful.
func (logs *Logs) streamFiles(files []os.FileInfo, w io.Writer) error {
	for _, fi := range files {
		file, err := os.Open(path.Join(logs.cfg.Directory, fi.Name()))
		if err != nil {
			return err
		}

		if err := logs.streamFile(file, 0, w); err != nil {
			return err
		}
	}

	return nil
}

// stream outputs the contents of a file with a given seek offset to a given writer.
func (logs *Logs) streamFile(file *os.File, offset int64, w io.Writer) error {
	defer func() {
		_ = file.Close()
	}()
	_, err := file.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		_, err := fmt.Fprintln(w, scanner.Text())
		if err != nil {
			return err
		}
	}

	return nil
}
