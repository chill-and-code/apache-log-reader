package logging

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type readerSuite struct {
	suite.Suite
	nowFunc func() time.Time
}

func (s *readerSuite) SetupSuite() {
	s.Require().NoError(os.RemoveAll(path.Dir(testDataDir)))
	s.Require().NoError(os.MkdirAll(testDataDir, 0777))

	// generate some logs and log files
	t, err := time.Parse(dateTimeFormat, "03/Mar/2022:02:45:00 +0000")
	s.Require().NoError(err)
	s.nowFunc = func() time.Time {
		return t
	}
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

func (s *readerSuite) TearDownSuite() {
	s.Require().NoError(os.RemoveAll(path.Dir(testDataDir)))
}

func (s *readerSuite) Test_NewReader_Success() {
	dir := "test/new-reader"
	s.Require().NoError(os.MkdirAll(dir, 0777))
	defer func() {
		s.Require().NoError(os.RemoveAll(dir))
	}()
	for i := 0; i < 5; i++ {
		s.createLogFile(dir, fmt.Sprintf("http-%d.log", i+1), fmt.Sprintf("log %d", i+1))
	}
	cfg := ReaderConfig{
		Directory:    dir,
		LastNMinutes: 3,
	}

	reader, err := NewReader(cfg)

	s.NoError(err)
	s.NotNil(reader)
	s.Equal(cfg, reader.cfg)
	s.Len(reader.filesInfo, 5)
}

func (s *readerSuite) Test_NewReader_Error() {
	cfg := ReaderConfig{
		Directory: "/path/to/nothing",
	}

	reader, err := NewReader(cfg)

	s.EqualError(err, "open /path/to/nothing: no such file or directory")
	s.Nil(reader)
}

func (s *readerSuite) Test_Read_Success() {
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
			ctx := context.Background()
			buf := &bytes.Buffer{}
			cfg := ReaderConfig{
				Directory:    testDataDir,
				LastNMinutes: test.lastNMinutes,
			}
			reader, err := NewReader(cfg)
			reader.nowFunc = s.nowFunc
			s.Require().NoError(err)

			err = reader.Read(ctx, buf)

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

func (s *readerSuite) Test_Read_OpenError() {
	ctx := context.Background()
	buf := &bytes.Buffer{}
	cfg := ReaderConfig{
		Directory: "/path/to/nothing",
	}

	reader := &Reader{
		nowFunc: s.nowFunc,
		cfg:     cfg,
		filesInfo: []os.FileInfo{
			fakeFile{name: "does-not-exist"},
		},
	}

	err := reader.Read(ctx, buf)

	s.EqualError(err, "open /path/to/nothing/does-not-exist: no such file or directory")
	s.Equal("", buf.String())
}

func (s *readerSuite) Test_Read_IndexTimeError() {
	dir := "test/index-time"
	s.Require().NoError(os.MkdirAll(dir, 0777))
	defer func() {
		s.Require().NoError(os.RemoveAll(dir))
	}()
	s.createLogFile(dir, "bad.log", "some invalid log")
	ctx := context.Background()
	buf := &bytes.Buffer{}
	cfg := ReaderConfig{
		Directory: dir,
	}
	reader, err := NewReader(cfg)
	reader.nowFunc = s.nowFunc
	s.Require().NoError(err)

	err = reader.Read(ctx, buf)

	s.EqualError(err, "line 'some invalid log': invalid log format")
	s.Equal("", buf.String())
}

func (s *readerSuite) createLogFile(dir, name, logs string) *os.File {
	file, err := os.Create(path.Join(dir, name))
	s.Require().NoError(err)
	_, err = file.WriteString(logs)
	s.Require().NoError(err)
	return file
}

func TestLogReader(t *testing.T) {
	suite.Run(t, new(readerSuite))
}

func BenchmarkLogReader(b *testing.B) {
	b.ReportAllocs()
}
