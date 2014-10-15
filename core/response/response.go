package response

import (
	"fmt"
)

/* Bookworm FTP control connection response */
type Response struct {
	status int
	message []byte
	multipleLines bool
}

/* Response builder */
func NewResponse(status int, message []byte, multipleLines bool) *Response {
	return &Response{status, message, multipleLines}
}

/* Response status getter */
func (r *Response) Status() int {
	return r.status
}

/* Response body getter */
func (r *Response) Message() string {
	return string(r.message)
}

/* Gives the ability to check the byte array. */
func (r *Response) ByteMessage() []byte {
	return r.message
}

/* Tells if the current Response has it's body composed of multiple lines */
func (r *Response) IsMultiLine() bool {
	return r.multipleLines
}

/* Response string serialization */
func (r *Response) String() string {
	s := " "
	if r.multipleLines {
		s = " - "
	}

	return fmt.Sprintf("%d%s%s", r.status, s, r.Message())
}
