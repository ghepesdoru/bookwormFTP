package response

/* Bookworm FTP control connection response */
type Response struct {
	status int
	message string
	multipleLines bool
}

/* Response builder */
func NewResponse(status int, message []byte, multipleLines bool) *Response {
	return &Response{status, string(message), multipleLines}
}

/* Response status getter */
func (r *Response) Status() int {
	return r.status
}

/* Response body getter */
func (r *Response) Message() string {
	return r.message
}

/* Tells if the current Response has it's body composed of multiple lines */
func (r *Response) IsMultiLine() bool {
	return r.multipleLines
}
