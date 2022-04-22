package logging

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	ipGroupName       = "ip"
	idGroupName       = "id"
	userGroupName     = "user"
	dateTimeGroupName = "datetime"
	requestGroupName  = "request"
	statusGroupName   = "status"
	sizeGroupName     = "size"
	dateTimeFormat    = "02/Jan/2006:15:04:05 -0700"
)

// NewFile wraps an os.File, creating a special apache common log format regex
// adding useful seek & search helper functions to easier work with log files.
// Here's an example of Apache Common Log format:
// 127.0.0.1 user-identifier frank [04/Mar/2022:05:30:00 +0000] "GET /api/endpoint HTTP/1.0" 500 123
func NewFile(file *os.File) File {
	ip := fmt.Sprintf(`(?P<%s>\S+)`, ipGroupName)
	id := fmt.Sprintf(`(?P<%s>\S+)`, idGroupName)
	user := fmt.Sprintf(`(?P<%s>\S+)`, userGroupName)
	datetime := fmt.Sprintf(`\[(?P<%s>[\w:/]+\s[+\-]\d{4})\]`, dateTimeGroupName)
	request := fmt.Sprintf(`"(?P<%s>\S+)\s?(\S+)?\s?(\S+)?"`, requestGroupName)
	status := fmt.Sprintf(`(?P<%s>\d{3}|-)`, statusGroupName)
	size := fmt.Sprintf(`(?P<%s>\d+|-)`, sizeGroupName)
	logFormat := fmt.Sprintf(`^%s %s %s %s %s %s %s$`, ip, id, user, datetime, request, status, size)

	return File{
		File:  file,
		regEx: regexp.MustCompile(logFormat),
	}
}

// File represents a wrapped structure around the os.File type
// providing additional constructs and helpers for working with log files
type File struct {
	*os.File
	regEx *regexp.Regexp
}

// IndexTime applies a binary search on a log file using Apache Common Log format, looking for
// the offset of the log that is within the lookup time (that took place within the last T time).
// offset >= 0 -> means an actual log line to begin reading logs at was found
// offset == -1 -> all the logs inside the log file are older than the lookup time T
func (file File) IndexTime(lookupTime time.Time) (int64, error) {
	var pos, prevPos int64
	scanLines := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		advance, token, err = bufio.ScanLines(data, atEOF)
		prevPos, pos = pos, pos+int64(advance)
		return
	}

	stat, err := file.Stat()
	if err != nil {
		return -1, err
	}
	top, bottom := int64(0), stat.Size()
	var prevLogTime time.Time
	for top <= bottom {
		middle := top + (bottom-top)/2
		_, err := file.Seek(middle, io.SeekStart)
		if err != nil {
			return -1, err
		}

		// reposition the middle to the beginning of the current line
		offset, err := file.seekLine()
		if err != nil {
			return -1, err
		}

		scanner := bufio.NewScanner(file)
		scanner.Split(scanLines)
		scanner.Scan()
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			// we'll consider empty line an EOF
			break
		}

		logTime, err := file.parseLogTime(line)
		if err != nil {
			return -1, err
		}

		if lookupTime.Sub(logTime) > 0 {
			// the starting log is way down (relative to the middle)
			// move down the top
			top = offset + (pos - prevPos)
		} else if prevLogTime.Sub(logTime) < 0 {
			// the starting log is way up (relative to the middle)
			// move up the bottom
			bottom = offset - (pos - prevPos)
		} else if lookupTime.Sub(prevLogTime) < 0 && offset != top {
			if lookupTime.Minute() == logTime.Minute() {
				return offset - (pos - prevPos), nil
			}
			return top, nil
		}

		if offset == top {
			if lookupTime.Minute() == logTime.Minute() || top == 0 {
				return top, nil
			}
			return offset - (pos - prevPos), nil
		}
		if offset == bottom {
			if lookupTime.Minute() > logTime.Minute() {
				return top, nil
			}
			return bottom, nil
		}

		prevLogTime = logTime
	}

	return -1, nil
}

// seekLine sets back the file cursor to the beginning of the closest line.
// Note: this function also repositions the internal file cursor at the closest new line offset.
func (file File) seekLine() (int64, error) {
	// check if we're already at the beginning of the file (offset 0)
	offset, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return -1, err
	}
	if offset == 0 {
		return file.Seek(0, io.SeekStart)
	}

	// traverse the file backwards till we reach a newline
	for {
		offset, err = file.Seek(-1, io.SeekCurrent)
		if err != nil {
			return -1, err
		}

		buf := make([]byte, 1)
		_, err = file.ReadAt(buf, offset)
		if err != nil {
			return -1, err
		}

		if offset == 0 {
			return file.Seek(0, io.SeekStart)
		}

		if buf[0] == '\n' {
			return file.Seek(1, io.SeekCurrent)
		}
	}
}

// parseLogTime parses a given Apache Common Log line and attempts to convert it into time.Time
// Here's an example of Apache Common Log format:
// 127.0.0.1 user-identifier frank [04/Mar/2022:05:30:00 +0000] "GET /api/endpoint HTTP/1.0" 500 123
func (file File) parseLogTime(logLine string) (time.Time, error) {
	matches := file.regEx.FindStringSubmatch(logLine)
	if len(matches) == 0 {
		return time.Time{}, fmt.Errorf("invalid log format on line '%s'", logLine)
	}

	var dateTime string
	for i, name := range file.regEx.SubexpNames() {
		if name == dateTimeGroupName {
			dateTime = matches[i]
			break
		}
	}
	if dateTime == "" {
		return time.Time{}, fmt.Errorf("invalid date format on line '%s'", logLine)
	}

	t, err := time.Parse(dateTimeFormat, dateTime)
	if err != nil {
		return time.Time{}, err
	}

	return t, nil
}
