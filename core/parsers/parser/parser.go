package parser

import(
	StatusCodes "github.com/ghepesdoru/bookwormFTP/core/codes"
	BaseParser "github.com/ghepesdoru/bookwormFTP/core/parsers/base"
	"github.com/ghepesdoru/bookwormFTP/core/response"
	"fmt"
)

const (
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
	var line, rawContent []byte
	var status, lineStatus, length, last, lastEOL, i, j, lineLength, count, linesCount int = -1, -1, len(raw), len(raw) - 1, 0, 0, 0, 0, 0, 0
	var multipleLines bool

	/* Loop over each character of the input */
	for i < length {
		c := raw[i]

		/* Greedy. Process once we hit the first non new line character. */
		if i == last || (BaseParser.IsNewLiner(c) && !BaseParser.IsNewLiner(raw[i + 1])) {
			line = BaseParser.TrimRight(raw[lastEOL:i])
			lastEOL = i

			/* Skip empty lines */
			lineLength = len(line)
			if lineLength == 0 {
				i += 1
				continue
			}

			lineStatus = -1
			count = 0
			for j = 0; j < lineLength; {
				s := line[j]

				/* Consume whitespaces, and new line */
				if !BaseParser.IsWhitespace(s) {
					count += 1

					if count <= 3 && StatusCodes.ByteIsNumber(s) {
						/* The line status has to be contained within the first 3 visible characters */
						if lineStatus == -1 {
							lineStatus = StatusCodes.ToInt([]byte{s})
						} else {
							lineStatus = lineStatus * 10 + StatusCodes.ToInt([]byte{s})
						}
					} else if !multipleLines && s == CONST_MultipleLinesResponseMark {
						/* Fill in case. The multiple lines flag will be determined by valid lines count. */
					} else {
						/* The current character is part of the message body. Break. */
						break
					}
				}

				j += 1
			}

			if (status == -1 && lineStatus != -1) || (lineStatus == -1 && status != -1) || (status == lineStatus && status != -1)  {
				/* First row of a response. Remember the response's status. */
				if status == -1 {
					if StatusCodes.IsValid(lineStatus) {
						status = lineStatus
					} else {
						/* Invalid response status code. */
						err = ERR_InvalidStatus
						break
					}
				}

				line = line[j:]

				/* Ignore empty lines */
				if j < (lineLength - 1) {
					rawContent = append(rawContent, append(line, []byte{10}...)...)
					linesCount += 1
				}

				/* Remember the consumed bytes */
				consumed = i
			} else {
				if status == -1 {
					/* Invalid line format */
					err = ERR_InvalidFormat
				}

				break
			}
		}

		i += 1
	}

	/* Ignore empty lines */
	if err == nil && (consumed == 0 || status == -1) {
		if len(BaseParser.Trim(raw)) == 0 {
			consumed = len(raw)
		}

		err = ERR_EmptyInput
	}

	if err == nil {
		if !multipleLines && linesCount > 1 {
			/* The current response contains multiple rows. No matter of the representation correctness */
			multipleLines = true
		}

		resp = response.NewResponse(status, rawContent, multipleLines)
	} else {
		resp = nil
	}

 	return
}
