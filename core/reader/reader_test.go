package reader

import (
	"testing"
	"io"
)

/* Definition of global variables */
var (
	testGlobals struct {
		pipeRead	*io.PipeReader
		pipeWrite	*io.PipeWriter
		reader		*Reader
	}
	WELCOME = []byte("200 Server Ready \r\n")
)

/* Initializes a new "server" to serve our FTP like messages */
func TestInitFakeServer(t *testing.T) {
	testGlobals.pipeRead, testGlobals.pipeWrite = io.Pipe()

	if testGlobals.pipeWrite == nil || testGlobals.pipeRead == nil {
		t.Log("Unable to establish test pipe", testGlobals.pipeRead, testGlobals.pipeWrite)
		t.Fail()
	}
}

/* Tests reader instantiation based on a local tcp connection - same machine */
func TestReaderInstantiation(t *testing.T) {
	reader := NewReader(testGlobals.pipeRead)

	if !reader.IsActive() {
		t.Fatal("Unable to initialize a Reader.")
	}

	if reader.Status() != STATUS[SIG_Done] {
		t.Fatal("Invalid reader status. Status: ", reader.Status() + "end of status")
	}

	testGlobals.reader = reader
}

/* Tests get capabilities of the current reader */
func TestGet(t *testing.T) {
	testGlobals.pipeWrite.Write(WELCOME)
	data := testGlobals.reader.Get()

	if len(data) == 0 {
		t.Fatal("No data read from pipe")
	}

	if string(data) != string(WELCOME) {
		t.Fatal("Invalid reading: ", string(data))
	}
}

/* Test read capabilities */
func TestRead(t *testing.T) {
	testGlobals.pipeWrite.Write(WELCOME) /* Ignored message */

	/* Block until the line to be ignored is read */
	for {
		data := testGlobals.reader.Peek()
		if len(data) > 0 {
			break
		}
	}

	go func () {
		testGlobals.pipeWrite.Write([]byte{'a'})
	}()

	data := testGlobals.reader.Read()

	if len(data) == 0 {
		t.Fatal("No data read from pipe")
	}

	if string(data) != "a" {
		t.Fatal("Invalid reading: ", string(data))
	}
}

/* Tests the reader stop reading functionality */
func TestReaderStop(t *testing.T) {
	testGlobals.reader.StopReading()

	if testGlobals.reader.IsActive() {
		t.Fatal("The reader activity flag was not updated successfully.")
	}
}
