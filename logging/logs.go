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

// LogsConfig represents the configuration to start the log readelogs.
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
// and streams it to a given writer.
func (logs *Logs) Print(ctx context.Context, w io.Writer) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		return logs.write(w)
	}
}

func (logs *Logs) logFileIndex() int {
	logFileIndex := -1
	for i, fi := range logs.filesInfo {
		if logs.nowMinusT().Sub(fi.ModTime()) <= 0 {
			logFileIndex = i
			break
		}
	}

	return logFileIndex
}

func (logs *Logs) write(w io.Writer) error {
	idx := logs.logFileIndex()
	if idx == -1 {
		return nil
	}

	filePath := path.Join(logs.cfg.Directory, logs.filesInfo[idx].Name())
	file, err := os.Open(filePath)
	defer func() { _ = file.Close() }()
	if err != nil {
		return err
	}

	offset, err := NewFile(file).IndexTime(logs.nowMinusT())
	if err != nil {
		return err
	}

	others := logs.filesInfo[idx+1 : len(logs.filesInfo)]
	if offset < 0 {
		if idx+1 >= len(logs.filesInfo) {
			return nil
		}

		fi := logs.filesInfo[idx+1]
		if logs.nowMinusT().Sub(fi.ModTime()) > 0 {
			return nil
		}
		return logs.stream(others, w)
	}

	_, err = file.Seek(offset, io.SeekStart)
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

	return logs.stream(others, w)
}

// stream reads from a given list of files and writes to a given writer.
// Because we need to preserve the order of the logs, and we want to also immediately stream to
// a given writer, we cannot use go routines. In a different scenario where order is not important
// that can of course be very useful.
func (logs *Logs) stream(files []os.FileInfo, w io.Writer) error {
	for _, fi := range files {
		file, err := os.Open(path.Join(logs.cfg.Directory, fi.Name()))
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

		_ = file.Close()
	}

	return nil
}
