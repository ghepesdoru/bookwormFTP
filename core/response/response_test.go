package response

import "testing"

func TestResponse(t *testing.T) {
	response := NewResponse(200, []byte("Server ready"), false)

	if response.Status() != 200 {
		t.Fatal("Invalid response status.")
	}

	if response.Message() != "Server ready" {
		t.Fatal("Invalid response message.")
	}

	if response.IsMultiLine() != false {
		t.Fatal("Invalid response multiple lines flag.")
	}
}
