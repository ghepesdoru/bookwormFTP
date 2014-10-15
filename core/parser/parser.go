package parser

import(
	StatusCodes "github.com/ghepesdoru/bookwormFTP/core/codes"
	"github.com/ghepesdoru/bookwormFTP/core/response"
	"fmt"
)

const (
	CONST_NewLine = 10
	CONST_NullChar = 0
	CONST_SpaceChar = 32
	CONST_CarriageReturn = 13
	CONST_Space = ' '
	CONST_MultipleLinesResponseMark = '-'
)

var (
	ERR_EmptyInput = fmt.Errorf("Empty response.")
	ERR_InvalidFormat = fmt.Errorf("Invalid line format.")
	ERR_InvalidStatus = fmt.Errorf("Invalid response line status. Wrongfully formatted multiple lines?")
	ERRF_ErrorParsing = "Parsing error: Error parsing input bytes: %s."
)

/* Parser type definition */
type Parser struct{
	responses []*response.Response
	errors []error
}

/* Empty Parser builder */
func NewParser() *Parser {
	return &Parser{[]*response.Response{}, []error{}}
}

func (r *Parser) ParseBlock(block []byte) {
	var length, consumed int = len(block), 0
	var resp *response.Response
	var err error

	for length > 0 {
		resp, consumed, err = r.parse(block)
		length -= consumed

		if err != nil {
			if err != ERR_EmptyInput {
				fmt.Println(fmt.Errorf(ERRF_ErrorParsing, err))
				r.errors = append(r.errors, err)
			}
		} else if resp != nil {
			r.responses = append(r.responses, resp)
		}

		if consumed == 0 {
			break
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
func (r *Parser) Errors() []error {
	return r.errors
}

/* Getter for the first encountered error */
func (r *Parser) LastError() error {
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
		/* Greedy. Process only once we hit a non new line character. */
		if (cIdx != length && (r.isNewLiner(c) && !r.isNewLiner(raw[cIdx + 1]))) || cIdx == length {
			line = raw[lastEOL:cIdx]
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
	if consumed == 0 || (status == -1 && err == nil) {
		if len(r.trim(raw)) == 0 {
			consumed = len(raw)
		}

		err = ERR_EmptyInput
	}

	if err == nil {
		resp = response.NewResponse(status, rawContent, multipleLines)
	} else {
		resp = nil
	}

 	return
}

func (r *Parser) isWhitespace(c byte) bool {
	if c == CONST_NullChar || c == CONST_SpaceChar || r.isNewLiner(c) {
		return true
	}

	return false
}

func (r *Parser) isNewLiner(c byte) bool {
	if c == CONST_NewLine || c == CONST_CarriageReturn {
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
	var end int = length

	for i := length; i >= 0; i -= 1 {
		if r.isWhitespace(line[i]) && end == i {
			end -= 1
		}
	}

	if end < (length + 1) {
		end += 1
	}

	return line[:end]
}
