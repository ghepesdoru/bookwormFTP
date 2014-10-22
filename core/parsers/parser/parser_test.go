package parser

import (
	"strings"
	"testing"
	Response "github.com/ghepesdoru/bookwormFTP/core/response"
)

var (
	EMPTYResponse = []byte{}
	SINGLEResponse = []byte{49, 50, 48, 32, 84, 104, 105, 115, 32, 105, 115, 32, 116, 104, 101, 32, 102, 116, 112, 45, 115, 101, 114, 118, 101, 114, 32, 111, 102, 32, 116, 104, 101, 32, 82, 73, 80, 69, 32, 78, 101, 116, 119, 111, 114, 107, 32, 67, 111, 111, 114, 100, 105, 110, 97, 116, 105, 111, 110, 32, 67, 101, 110, 116, 114, 101, 32, 40, 78, 67, 67, 41, 46, 13, 10}
	SINGLEResponseMulti = []byte{50, 50, 48, 45, 13, 10, 84, 104, 105, 115, 32, 105, 115, 32, 116, 104, 101, 32, 102, 116, 112, 45, 115, 101, 114, 118, 101, 114, 32, 111, 102, 32, 116, 104, 101, 32, 82, 73, 80, 69, 32, 78, 101, 116, 119, 111, 114, 107, 32, 67, 111, 111, 114, 100, 105, 110, 97, 116, 105, 111, 110, 32, 67, 101, 110, 116, 114, 101, 32, 40, 78, 67, 67, 41, 46, 13, 10, 80, 108, 101, 97, 115, 101, 32, 114, 101, 112, 111, 114, 116, 32, 112, 114, 111, 98, 108, 101, 109, 115, 32, 116, 111, 58, 32, 119, 101, 98, 109, 97, 115, 116, 101, 114, 32, 97, 116, 32, 114, 105, 112, 101, 46, 110, 101, 116, 13, 10, 70, 84, 80, 32, 83, 101, 114, 118, 101, 114, 32, 114, 101, 97, 100, 13, 10}
	Response_ServerReady = Response.NewResponse(120, []byte{84, 104, 105, 115, 32, 105, 115, 32, 116, 104, 101, 32, 102, 116, 112, 45, 115, 101, 114, 118, 101, 114, 32, 111, 102, 32, 116, 104, 101, 32, 82, 73, 80, 69, 32, 78, 101, 116, 119, 111, 114, 107, 32, 67, 111, 111, 114, 100, 105, 110, 97, 116, 105, 111, 110, 32, 67, 101, 110, 116, 114, 101, 32, 40, 78, 67, 67, 41, 46, 10}, false)
	Response_Welcome = Response.NewResponse(220, []byte{84, 104, 105, 115, 32, 105, 115, 32, 116, 104, 101, 32, 102, 116, 112, 45, 115, 101, 114, 118, 101, 114, 32, 111, 102, 32, 116, 104, 101, 32, 82, 73, 80, 69, 32, 78, 101, 116, 119, 111, 114, 107, 32, 67, 111, 111, 114, 100, 105, 110, 97, 116, 105, 111, 110, 32, 67, 101, 110, 116, 114, 101, 32, 40, 78, 67, 67, 41, 46, 10, 80, 108, 101, 97, 115, 101, 32, 114, 101, 112, 111, 114, 116, 32, 112, 114, 111, 98, 108, 101, 109, 115, 32, 116, 111, 58, 32, 119, 101, 98, 109, 97, 115, 116, 101, 114, 32, 97, 116, 32, 114, 105, 112, 101, 46, 110, 101, 116, 10, 70, 84, 80, 32, 83, 101, 114, 118, 101, 114, 32, 114, 101, 97, 100, 10}, true)
)

func TestEmptyResponse(t *testing.T) {
	parser := NewParser()
	parser.ParseBlock(EMPTYResponse)

	/* Parsing an empty block does not trigger errors */
	if parser.HasErrors() {
		t.Fatal("Parsing of empty block triggering errors.", parser.LastError())
	}
}

func test(response *Response.Response, testAgainst *Response.Response, identifier string, t *testing.T) {
	if response.Status() != testAgainst.Status() {
		t.Fatal("Invalid " + identifier + " response parsed status.", response.Status())
	}

	if response.Message() != testAgainst.Message() {
		t.Fatal("Invalid " + identifier + " response parsed message", response.ByteMessage(), testAgainst.ByteMessage())
	}

	if response.IsMultiLine() != testAgainst.IsMultiLine() {
		t.Fatal("Invalid " + identifier + " response is multilined", response.IsMultiLine())
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
		t.Fatal("Invalid responses count for single response block after fetching the response.", parser.Length())
	}

	test(response, Response_ServerReady, "single response single line", t)
}

func TestSingleResponseWithMultipleLines(t *testing.T) {
	parser := NewParser()
	parser.ParseBlock(SINGLEResponseMulti)

	if parser.HasErrors() {
		t.Fatal("Parsing of valid single response with multiple lines block triggering errors.", parser.LastError())
	}

	if parser.Length() != 1 {
		t.Fatal("Invalid responses count for single response with multiple lines block.", parser.Length())
	}

	response := parser.Get()

	if parser.Length() != 0 {
		t.Fatal("Invalid responses count for single response with multiple lines block after fetching the response.", parser.Length())
	}

	test(response, Response_Welcome, "single response multiple lines", t)
}

func TestMultipleMessages(t *testing.T) {
	parser := NewParser()
	both := append(SINGLEResponse, []byte{13, 10}...)
	both = append(SINGLEResponse, SINGLEResponseMulti...)
	parser.ParseBlock(both)

	if parser.HasErrors() {
		t.Fatal("Parsing of multiple messages block triggering errors.", parser.LastError())
	}

	if parser.Length() != 2 {
		t.Fatal("Invalid responses count for 2 responses context.", parser.Length())
	}

	response1 := parser.Get()
	response2 := parser.Get()

	if parser.Length() != 0 {
		t.Fatal("Invalid response count for 2 responses context after fetching 2.", parser.Length())
	}

	test(response1, Response_ServerReady, "first of 2 responses - single lined", t)
	test(response2, Response_Welcome, "second of 2 responses - multiple lined", t)
}

func TestInvalidFormat(t *testing.T) {
	parser := NewParser()
	parser.ParseBlock([]byte(`This is
	a random
	text. It shold generate a ERR_InvalidFormat`))

	if parser.HasErrors() {
		if !strings.Contains(parser.LastError().Error(), ERR_InvalidFormat.Error()) {
			t.Fatal("Invalid parsing error type for invalid format input.", parser.LastError())
		}
	} else {
		t.Fatal("Invalid parsing. Invalid format input does not generate error.", parser.Get())
	}
}

func TestInvalidResponseLineStatus(t *testing.T) {
	parser := NewParser()
	parser.ParseBlock([]byte(`999 Unknown message status.`))

	if parser.HasErrors() {
		if !strings.Contains(parser.LastError().Error(), ERR_InvalidStatus.Error()) {
			t.Fatal("Invalid parsing error type for invalid status input.", parser.LastError())
		}
	} else {
		t.Fatal("Invalid parsing. Invalid status input does not generate error.", parser.Get())
	}
}
