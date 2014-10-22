package logger

import (
	"testing"
	"os"
	"io/ioutil"
	"path/filepath"
	"fmt"
)

const (
	Sample = "Test"
)

var (
	TMP_FILE 	*os.File
	TMP_FileName string
	FILE		*os.File
)

func test(level int, message string, t *testing.T) {
	read := append(LoggingHeaders[level], flush()...)

	message = string(LoggingHeaders[level]) + message

	if string(read) != message {
		t.Fatal("Invalid reading of logged data at level", level, string(read), message)
	}
}

func capture() {
	TMP_FileName := filepath.Join(os.TempDir(), "stdout")
	FILE = os.Stdout
	TMP_FILE, _ = os.Create(TMP_FileName)
	os.Stdout = TMP_FILE
}

func flush() []byte {
	TMP_FILE.Close()
	os.Stdout = FILE
	out, _ := ioutil.ReadFile(TMP_FileName)
	fmt.Println("Tmp file contains: ", string(out))
	return out
}

func TestLog(t *testing.T) {
	logger := NewSimpleLogger()

	capture()
	logger.Information(Sample)
	test(LOG_Information, Sample, t)

	capture()
	logger.Warning(Sample)
	test(LOG_Warning, Sample, t)

	capture()
	logger.Error(Sample)
	test(LOG_Error, Sample, t)

	capture()
	logger.Critical(Sample)
	test(LOG_Critical, Sample, t)
}

