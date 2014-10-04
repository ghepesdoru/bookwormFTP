package responseParser

import(
	StatusCodes "github.com/ghepesdoru/bookwormFTP/core/codes"
	"github.com/ghepesdoru/bookwormFTP/core/response"
	"bytes"
	"fmt"
)

const (
	CONST_NewLine = '\n'
	CONST_NullChar = '\000'
	CONST_CarriageReturn = '\r'
	CONST_Space = ' '
	CONST_MultipleLinesResponseMark = '-'
)

var (
	ERR_EmptyInput = fmt.Errorf("Empty response.")
	ERR_InvalidFormat = fmt.Errorf("Invalid line format.")
	ERR_InvalidStatus = fmt.Errorf("Invalid response line status. Wrongfully formatted multiple lines?")
)

/* Parser type definition */
type Parser struct{
	responses []*response.Response
	errors []*error
}

/* Empty Parser builder */
func NewParser() *Parser {
	return &Parser{[]*response.Response{}, []*error{}}
}

func (r *Parser) ParseBlock(block []byte) {
	var length, consumed int = len(block), 0
	var resp *response.Response
	var err error

	block = bytes.TrimSpace(block)

	for length > 0 {
		resp, consumed, err = r.parse(block)
		length -= consumed

		if err != nil {
			if err != ERR_EmptyInput {
				fmt.Println("Error while parsing: ", err)
				r.errors = append(r.errors, &err)
			}
		} else {
			r.responses = append(r.responses, resp)
		}

		/* Reduce the parsed block with the parsed quantity */
		block = block[consumed:]
	}
}

/* Returns the first response in the responses list */
func (r *Parser) Get() *response.Response {
	var resp *response.Response = nil

	if r.Length() > 0 {
		resp = r.responses[0]
		r.responses = r.responses[1:]
	}

	return resp
}

/* Returns the number of not consumed parsed responses */
func (r *Parser) Length() int {
	return len(r.responses)
}

/* Checks if the Parser has encountered an error */
func (r *Parser) HasErrors() bool {
	return len(r.errors) != 0
}

/* Returns all encountered errors while parsing */
func (r *Parser) Errors() []*error {
	return r.errors
}

/* Getter for the first encountered error */
func (r *Parser) LastError() *error {
	if r.HasErrors() {
		return r.errors[0]
	}

	return nil
}

/* Response parser utility. Parses one response at a time, and returns the parsed response, number of consumed bytes,
 and any errors if required */
func (r *Parser) parse(raw []byte) (resp *response.Response, consumed int, err error) {
	var line, rawStatus, rawContent []byte
	var charConsumed, status, lineStatus, length, lastEOL int = -1, -1, 0, len(raw) - 1, 0
	var multipleLines bool

	/* Loop over each character of input source */
	for cIdx, c := range raw {
		if c == CONST_NewLine || cIdx == length {
			/* End of line/input found, generate a line from the last line ending position */
			if cIdx > 0 && raw[cIdx - 1] == CONST_CarriageReturn {
				/* Exclude the carriage return from the current line */
				line = raw[lastEOL:cIdx - 1]
			} else {
				line = raw[lastEOL:cIdx]
			}

			line = r.trim(line)
			lastEOL = cIdx

			/* Reset the line status buffer */
			rawStatus = []byte{}

			if len(line) != 0 {
				/* Look over the current line content's, extracting the status code, and eating spaces */
				for i, c := range line {
					if i < 3 && StatusCodes.ByteIsNumber(c) {
						rawStatus = append(rawStatus, c)
						charConsumed = i
					} else if !multipleLines && c == CONST_MultipleLinesResponseMark {
						/* Check for multiple lines response mark */
						multipleLines = true
						charConsumed = i + 1 /* Exclude the multiple line token from the output */
					} else if c != CONST_Space {
						charConsumed = i
						break
					}
				}

				/* Convert status to a number */
				lineStatus = StatusCodes.ToInt(rawStatus)

				/* Convert multiple lines with the same status to the same response message */
				if (lineStatus == -1 && len(rawStatus) == 0) || status == lineStatus || status == -1 {
					/* Remember the current index for the consumed characters in input */
					consumed = cIdx

					/* Check for invalid format responses */
					if status == -1 && lineStatus == -1 {
						err = ERR_InvalidFormat
					} else {
						if status < 0 {
							/* First time assignment */
							status = lineStatus
						}

						if StatusCodes.IsValid(status) {
							/* Check if this is a wrong formatted multiple lines response */
							if !multipleLines && len(rawContent) > 0 {
								multipleLines = true
							}

							rawContent = append(rawContent, line[charConsumed:]...)
							rawContent = append(rawContent, CONST_NewLine)
						} else {
							fmt.Println(string(line))
							err = ERR_InvalidStatus
						}
					}
				} else {
					/* Passed the boundaries of the previous response. Break here! */
					break
				}
			}
		} else {
			consumed = cIdx
		}
	}

	/* Ignore empty lines */
	if consumed == 0 {
		if len(r.trim(raw)) == 0 {
			consumed = len(raw)
			err = ERR_EmptyInput
		}
	} else if status == -1 && err == nil {
		err = ERR_EmptyInput
	}

	resp = response.NewResponse(status, rawContent, multipleLines)
	return
}

func (r *Parser) isWhitespace(c byte) bool {
	if c == CONST_NewLine || c == CONST_NullChar || c == CONST_NewLine || c == CONST_CarriageReturn {
		return true
	}

	return false
}

func (r *Parser) trim (line []byte) []byte {
	line = r.trimLeft(line)
	return r.trimRight(line)
}

func (r *Parser) trimLeft(line []byte) []byte {
	var start int = 0

	for i, c := range line {
		if r.isWhitespace(c) && i == start {
			/* At start of the string */
			start += 1
		}
	}

	return line[start:]
}

func (r *Parser) trimRight(line []byte) []byte {
	var length int = len (line) - 1
	var end int = length + 1

	for i := length; i > -1; i -= 1 {
		if r.isWhitespace(line[i]) && end == i {
			end -= 1
		}
	}

	return line[:end]
}
