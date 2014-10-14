package parser

import (
	"testing"
	Response "github.com/ghepesdoru/bookwormFTP/core/response"
)

var (
	EMPTYResponse = []byte{}
	SINGLEResponse = []byte("200 Server Ready \r\n")
	SINGLEResponseMulti = []byte("200 - \r\nThis is the TestFTP Server. Welcome. \r\n200")
	Response_ServerReady = Response.NewResponse(200, []byte{83, 101, 114, 118, 101, 114, 32, 82, 101, 97, 100, 121, 32, 10}, false)
	Response_Welcome = Response.NewResponse(200, []byte{84, 104, 105, 115, 32, 105, 115, 32, 116, 104, 101, 32, 84, 101, 115, 116, 70, 84, 80, 32, 83, 101, 114, 118, 101, 114, 46, 32, 87, 101, 108, 99, 111, 109, 101, 46}, true)
)

func TestEmptyResponse(t *testing.T) {
	parser := NewParser()
	parser.ParseBlock(EMPTYResponse)

	/* Parsing an empty block does not trigger errors */
	if parser.HasErrors() {
		t.Fatal("Parsing of empty block triggering errors.", parser.LastError())
	}
}

func test(response *Response.Response, testAgainst *Response.Response, t *testing.T) {
	if response.Status() != Response_ServerReady.Status() {
		t.Fatal("Invalid response parsed status.", testAgainst.Status())
	}

	if response.Message() != Response_ServerReady.Message() {
		t.Fatal("Invalid response parsed message", testAgainst.Message())
	}

	if response.IsMultiLine() != Response_ServerReady.IsMultiLine() {
		t.Fatal("Invalid response parsed message", testAgainst.IsMultiLine())
	}
}

func TestSingleResponse(t *testing.T) {
	parser := NewParser()
	parser.ParseBlock(SINGLEResponse)

	if parser.HasErrors() {
		t.Fatal("Parsing of valid single response block triggering errors.", parser.LastError())
	}

	if parser.Length() != 1 {
		t.Fatal("Invalid responses count for single response block.", parser.Length())
	}

	response := parser.Get()

	if parser.Length() != 0 {
		t.Fatal("Invalid responses count for single response block after fetching the single response.", parser.Length())
	}

	test(response, Response_ServerReady, t)
}

func TestSingleResponseWithMultipleLines(t *testing.T) {
	parser := NewParser()
	parser.ParseBlock(SINGLEResponseMulti)

	if parser.HasErrors() {
		t.Fatal("Parsing of valid single response block triggering errors.", parser.LastError())
	}

	if parser.Length() != 1 {
		t.Fatal("Invalid responses count for single response block.", parser.Length())
	}

	response := parser.Get()

	if parser.Length() != 0 {
		t.Fatal("Invalid responses count for single response block after fetching the single response.", parser.Length())
	}

	test(response, Response_Welcome, t)
}
