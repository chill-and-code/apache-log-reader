package logging

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type logsSuite struct {
	suite.Suite
	testTime time.Time
}

func (s *logsSuite) SetupSuite() {
	s.Require().NoError(os.RemoveAll(path.Dir(testDataDir)))
	s.Require().NoError(os.MkdirAll(testDataDir, 0777))

	// generate a few log files with some dummy logs for testing
	t, err := time.Parse(dateTimeFormat, "03/Mar/2022:02:45:00 +0000")
	s.Require().NoError(err)
	s.testTime = t
	now := t.Add(-time.Minute)
	numOfFiles := 3
	numOfLogs := 3
	for i := 0; i < numOfFiles; i++ {
		logs := fmt.Sprintf(`127.0.0.1 user-identifier frank [%v] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [%v] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [%v] "GET /api/endpoint HTTP/1.0" 500 123
`,
			now.Add(-time.Duration(numOfFiles-i+4)*20*time.Second).Format(dateTimeFormat),
			now.Add(-time.Duration(numOfFiles-i+3)*20*time.Second).Format(dateTimeFormat),
			now.Add(-time.Duration(numOfFiles-i+2)*20*time.Second).Format(dateTimeFormat),
		)

		s.createLogFile(testDataDir, fmt.Sprintf("http-%d.log", i+1), logs)
		err = os.Chtimes(path.Join(testDataDir, fmt.Sprintf("http-%d.log", i+1)), now, now)
		s.Require().NoError(err)
		now = now.Add(time.Duration(numOfLogs)*20*time.Second + 20*time.Second)
	}
}

func (s *logsSuite) TearDownSuite() {
	s.Require().NoError(os.RemoveAll(path.Dir(testDataDir)))
}

func (s *logsSuite) Test_NewLogs_Success() {
	dir := "test/logs"
	s.Require().NoError(os.MkdirAll(dir, 0777))
	defer func() {
		s.Require().NoError(os.RemoveAll(dir))
	}()
	for i := 0; i < 5; i++ {
		s.createLogFile(dir, fmt.Sprintf("http-%d.log", i+1), fmt.Sprintf("log %d", i+1))
	}
	cfg := LogsConfig{
		Directory:    dir,
		LastNMinutes: 3,
	}

	logs, err := NewLogs(cfg)

	s.NoError(err)
	s.NotNil(logs)
	s.Equal(cfg, logs.cfg)
	s.Len(logs.filesInfo, 5)
}

func (s *logsSuite) Test_NewLogs_Error() {
	cfg := LogsConfig{
		Directory: "/path/to/nothing",
	}

	logs, err := NewLogs(cfg)

	s.EqualError(err, "open /path/to/nothing: no such file or directory")
	s.Nil(logs)
}

func (s *logsSuite) Test_Print_Success() {
	tests := []struct {
		name         string
		lastNMinutes int
		expectedLogs string
	}{
		{
			name:         "Last Minute",
			lastNMinutes: 1,
			expectedLogs: `127.0.0.1 user-identifier frank [03/Mar/2022:02:43:20 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:43:40 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:44:00 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:45:00 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:45:20 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:45:40 +0000] "GET /api/endpoint HTTP/1.0" 500 123
`,
		},
		{
			name:         "Last Two Minutes",
			lastNMinutes: 2,
			expectedLogs: `127.0.0.1 user-identifier frank [03/Mar/2022:02:43:20 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:43:40 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:44:00 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:45:00 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:45:20 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:45:40 +0000] "GET /api/endpoint HTTP/1.0" 500 123
`,
		},
		{
			name:         "Last Three Minutes",
			lastNMinutes: 3,
			expectedLogs: `127.0.0.1 user-identifier frank [03/Mar/2022:02:42:00 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:42:20 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:43:20 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:43:40 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:44:00 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:45:00 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:45:20 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:45:40 +0000] "GET /api/endpoint HTTP/1.0" 500 123
`,
		},
		{
			name:         "Last Four Minutes",
			lastNMinutes: 4,
			expectedLogs: `127.0.0.1 user-identifier frank [03/Mar/2022:02:41:40 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:42:00 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:42:20 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:43:20 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:43:40 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:44:00 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:45:00 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:45:20 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:45:40 +0000] "GET /api/endpoint HTTP/1.0" 500 123
`,
		},
		{
			name:         "Last Five Hours",
			lastNMinutes: 60 * 5,
			expectedLogs: `127.0.0.1 user-identifier frank [03/Mar/2022:02:41:40 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:42:00 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:42:20 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:43:20 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:43:40 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:44:00 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:45:00 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:45:20 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [03/Mar/2022:02:45:40 +0000] "GET /api/endpoint HTTP/1.0" 500 123
`,
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			buf := &bytes.Buffer{}
			cfg := LogsConfig{
				Directory:    testDataDir,
				LastNMinutes: test.lastNMinutes,
			}
			logs, err := NewLogs(cfg)
			logs.nowMinusT = func() time.Time {
				return s.testTime.Add(-time.Duration(cfg.LastNMinutes) * time.Minute)
			}
			s.Require().NoError(err)

			err = logs.Print(buf)

			s.NoError(err)
			s.Equal(test.expectedLogs, buf.String())
		})
	}
}

type fakeFile struct {
	name string
}

func (f fakeFile) Name() string       { return f.name }
func (f fakeFile) Size() int64        { return 1024 }
func (f fakeFile) Mode() fs.FileMode  { return os.ModePerm }
func (f fakeFile) ModTime() time.Time { return time.Now() }
func (f fakeFile) IsDir() bool        { return false }
func (f fakeFile) Sys() interface{}   { return nil }

func (s *logsSuite) Test_Print_OpenError() {
	buf := &bytes.Buffer{}
	cfg := LogsConfig{
		Directory: "/path/to/nothing",
	}

	logs := &Logs{
		nowMinusT: func() time.Time {
			return s.testTime
		},
		cfg: cfg,
		filesInfo: []os.FileInfo{
			fakeFile{name: "does-not-exist"},
		},
	}

	err := logs.Print(buf)

	s.EqualError(err, "open /path/to/nothing/does-not-exist: no such file or directory")
	s.Equal("", buf.String())
}

func (s *logsSuite) Test_Print_IndexTimeError() {
	dir := "test/index-time"
	s.Require().NoError(os.MkdirAll(dir, 0777))
	defer func() {
		s.Require().NoError(os.RemoveAll(dir))
	}()
	s.createLogFile(dir, "bad.log", "some invalid log")
	buf := &bytes.Buffer{}
	cfg := LogsConfig{
		Directory: dir,
	}
	logs, err := NewLogs(cfg)
	logs.nowMinusT = func() time.Time {
		return s.testTime
	}
	s.Require().NoError(err)

	err = logs.Print(buf)

	s.EqualError(err, "invalid log format on line 'some invalid log'")
	s.Equal("", buf.String())
}

func (s *logsSuite) createLogFile(dir, name, logs string) *os.File {
	file, err := os.Create(path.Join(dir, name))
	s.Require().NoError(err)
	_, err = file.WriteString(logs)
	s.Require().NoError(err)
	return file
}

func TestLogs(t *testing.T) {
	suite.Run(t, new(logsSuite))
}
