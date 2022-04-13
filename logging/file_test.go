package logging

import (
	"bufio"
	"io"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	testDataDir  = "test/testdata"
	benchDataDir = "bench/testdata"
)

type fileSuite struct {
	suite.Suite
}

func (s *fileSuite) SetupSuite() {
	s.Require().NoError(os.RemoveAll(path.Dir(testDataDir)))
	s.Require().NoError(os.MkdirAll(testDataDir, 0777))
}

func (s *fileSuite) TearDownSuite() {
	s.Require().NoError(os.RemoveAll(path.Dir(testDataDir)))
}

func (s *fileSuite) Test_NewFile() {
	f, err := os.CreateTemp(testDataDir, "*")
	defer func() { s.Require().NoError(f.Close()) }()
	s.Require().NoError(err)

	file := NewFile(f)

	s.NotNil(file)
	s.NotNil(file.File)
	s.NotNil(file.regEx)
}

func (s *fileSuite) Test_IndexTime_Success() {
	logs := `127.0.0.1 user-identifier frank [07/Mar/2022:02:39:32 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [07/Mar/2022:02:39:42 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [07/Mar/2022:02:39:52 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [07/Mar/2022:02:40:02 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [07/Mar/2022:02:40:12 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [07/Mar/2022:02:40:22 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [07/Mar/2022:02:40:32 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [07/Mar/2022:02:40:42 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [07/Mar/2022:02:40:52 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [07/Mar/2022:02:41:02 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [07/Mar/2022:02:41:12 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [07/Mar/2022:02:41:22 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [07/Mar/2022:02:41:32 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [07/Mar/2022:02:41:42 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [07/Mar/2022:02:41:52 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [07/Mar/2022:02:42:02 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [07/Mar/2022:02:42:12 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [07/Mar/2022:02:42:22 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [07/Mar/2022:02:42:32 +0000] "GET /api/endpoint HTTP/1.0" 500 123
127.0.0.1 user-identifier frank [07/Mar/2022:02:42:42 +0000] "GET /api/endpoint HTTP/1.0" 500 123`
	now, err := time.Parse(dateTimeFormat, "07/Mar/2022:02:43:00 +0000")
	s.Require().NoError(err)
	f := s.createLogs(logs)
	defer func() { s.Require().NoError(f.Close()) }()
	file := NewFile(f)
	s.NotNil(file)
	tests := []struct {
		name           string
		expectedOffset int64
		expectedLog    string
		timeLookup     time.Time
	}{
		{
			name:           "Last 4 Minutes",
			timeLookup:     now.Add(-4 * time.Minute),
			expectedOffset: 0,
			expectedLog:    `127.0.0.1 user-identifier frank [07/Mar/2022:02:39:32 +0000] "GET /api/endpoint HTTP/1.0" 500 123`,
		},
		{
			name:           "Last 3 Minutes",
			timeLookup:     now.Add(-3 * time.Minute),
			expectedOffset: 294,
			expectedLog:    `127.0.0.1 user-identifier frank [07/Mar/2022:02:40:02 +0000] "GET /api/endpoint HTTP/1.0" 500 123`,
		},
		{
			name:           "Last 2 Minutes",
			timeLookup:     now.Add(-2 * time.Minute),
			expectedOffset: 882,
			expectedLog:    `127.0.0.1 user-identifier frank [07/Mar/2022:02:41:02 +0000] "GET /api/endpoint HTTP/1.0" 500 123`,
		},
		{
			name:           "Last Minute",
			timeLookup:     now.Add(-time.Minute),
			expectedOffset: 1470,
			expectedLog:    `127.0.0.1 user-identifier frank [07/Mar/2022:02:42:02 +0000] "GET /api/endpoint HTTP/1.0" 500 123`,
		},
		{
			name:           "Last 2 Days From Beginning",
			timeLookup:     now.Add(-2 * time.Hour * 24),
			expectedOffset: 0,
			expectedLog:    `127.0.0.1 user-identifier frank [07/Mar/2022:02:39:32 +0000] "GET /api/endpoint HTTP/1.0" 500 123`,
		},
		{
			name:           "Future Minute No Logs",
			timeLookup:     now.Add(time.Minute),
			expectedOffset: -1,
			expectedLog:    ``,
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			offset, err := file.IndexTime(test.timeLookup)
			log := s.readLogAt(f, offset)

			s.NoError(err)
			s.Equal(test.expectedOffset, offset)
			s.Equal(test.expectedLog, log)
		})
	}
}

func (s *fileSuite) Test_IndexTime_Error() {
	f := s.createLogs("some invalid log line\n")
	defer func() { s.Require().NoError(f.Close()) }()
	file := NewFile(f)
	s.NotNil(file)

	lookupTime := time.Now().UTC().Add(-1 * time.Minute)
	offset, err := file.IndexTime(lookupTime)

	s.EqualError(err, "line 'some invalid log line': invalid log format")
	s.Equal(int64(-1), offset)
}

func (s *fileSuite) Test_seekLine() {
	data := "some\ntest\nstring\n"
	f := s.createLogs(data)
	defer func() { s.Require().NoError(f.Close()) }()

	_, err := f.Seek(8, io.SeekStart)
	s.NoError(err)
	file := NewFile(f)
	s.NotNil(file)

	tests := []struct {
		name           string
		lines          int64
		whence         int
		expectedOffset int64
	}{
		{
			name:           "Line Zero CurrentLine",
			lines:          0,
			whence:         io.SeekCurrent,
			expectedOffset: 5,
		},
		{
			name:           "LinesGreaterThanZero SeekCurrent",
			lines:          3,
			whence:         io.SeekCurrent,
			expectedOffset: int64(len(data)),
		},
		{
			name:           "LinesGreaterThanZero SeekEnd",
			lines:          3,
			whence:         io.SeekEnd,
			expectedOffset: int64(len(data)),
		},
		{
			name:           "LinesGreaterThanZero SeekStart LastLine",
			lines:          3,
			whence:         io.SeekStart,
			expectedOffset: int64(len(data)),
		},
		{
			name:           "LineGreaterThanZero SeekStart SecondLine",
			lines:          2,
			whence:         io.SeekStart,
			expectedOffset: 10,
		},
		{
			name:           "NegativeLines SeekStart SecondLine",
			lines:          -2,
			whence:         io.SeekStart,
			expectedOffset: 10,
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			offset, err := file.seekLine(test.lines, test.whence)

			s.NoError(err)
			s.Equal(test.expectedOffset, offset)
		})
	}
}

func (s *fileSuite) Test_parseLogTime_Success() {
	log := `127.0.0.1 user-identifier frank [04/Mar/2022:05:30:00 +0000] "GET /api/endpoint HTTP/1.0" 500 123`
	expectedTime, err := time.Parse(dateTimeFormat, "04/Mar/2022:05:30:00 +0000")
	s.Require().NoError(err)
	file := NewFile(nil)
	s.NotNil(file)

	t, err := file.parseLogTime(log)

	s.NoError(err)
	s.True(t.Equal(expectedTime))
}

func (s *fileSuite) Test_parseLogTime_Error() {
	file := NewFile(nil)
	s.NotNil(file)
	tests := []struct {
		name        string
		log         string
		expectedErr string
	}{
		{
			name:        "Empty LogLine",
			log:         "",
			expectedErr: "line '': invalid log format",
		},
		{
			name:        "Invalid LogLine",
			log:         "this log line is not valid",
			expectedErr: "line 'this log line is not valid': invalid log format",
		},
		{
			name:        "Invalid DateFormat",
			log:         `127.0.0.1 user-identifier frank [36/Mar/2022:05:30:00 +0000] "GET /api/endpoint HTTP/1.0" 500 123`,
			expectedErr: `parsing time "36/Mar/2022:05:30:00 +0000": day out of range`,
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			t, err := file.parseLogTime(test.log)

			s.EqualError(err, test.expectedErr)
			s.True(t.IsZero())
		})
	}
}

// createLogs stores incoming logs in a temporary file
// make sure the incoming logs end with a newline
// otherwise future scans might hang.
func (s *fileSuite) createLogs(logs string) *os.File {
	file, err := os.CreateTemp(testDataDir, "*-http.log")
	s.Require().NoError(err)
	_, err = file.WriteString(logs)
	s.Require().NoError(err)
	return file
}

// readLogAt reads 1 log line at a given offset from a given file
func (s *fileSuite) readLogAt(file *os.File, offset int64) string {
	if offset < 0 {
		return ""
	}

	_, err := file.Seek(offset, io.SeekStart)
	s.Require().NoError(err)

	scanner := bufio.NewScanner(file)
	scanner.Scan()
	return scanner.Text()
}

func TestLogFile(t *testing.T) {
	suite.Run(t, new(fileSuite))
}

// Generate the big log file to be able to benchmark properly
// and make sure to store it inside benchDataDir
func BenchmarkSearch(b *testing.B) {
	// log-generator stores the big data in http-1.log
	f, err := os.Open(path.Join(benchDataDir, "http-1.log"))
	defer func() { require.NoError(b, f.Close()) }()
	require.NoError(b, err)
	file := NewFile(f)
	require.NotNil(b, file)
	b.ResetTimer()

	// we don't care about the offset, we only want to benchmark
	// and check for execution time and memory footprint
	for i := 0; i < b.N; i++ {
		lookupTime := time.Now().UTC().Add(-time.Duration(i) * time.Minute)
		_, err = file.IndexTime(lookupTime)
		require.NoError(b, err)
	}

	b.ReportAllocs()
}
