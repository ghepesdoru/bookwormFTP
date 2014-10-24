package base

import (
	"bytes"
)

/* Basic Parsing functionality, reused in all parsers. */
const(
	CONST_NewLine = 10
	CONST_NullChar = 0
	CONST_SpaceChar = 32
	CONST_CarriageReturn = 13
)

func IsWhitespace(c byte) bool {
	if c == CONST_NullChar || c == CONST_SpaceChar || IsNewLiner(c) {
		return true
	}

	return false
}

func IsNewLiner(c byte) bool {
	if c == CONST_NewLine || c == CONST_CarriageReturn {
		return true
	}

	return false
}

func SplitLines(bulk []byte) (lines [][]byte) {
	lines = bytes.Split(bulk, []byte{CONST_NewLine, CONST_CarriageReturn})
	if len(lines) == 1 {
		lines = bytes.Split(bulk, []byte{CONST_NewLine})
	}

	return lines
}

func Trim (line []byte) []byte {
	line = TrimLeft(line)
	return TrimRight(line)
}

func TrimLeft(line []byte) []byte {
	var start int = 0

	for i, c := range line {
		if IsWhitespace(c) && i == start {
			/* At start of the string */
			start += 1
		}
	}

	return line[start:]
}

func TrimRight(line []byte) []byte {
	var length int = len(line) - 1
	var end int = length

	for i := length; i >= 0; i -= 1 {
		if IsWhitespace(line[i]) && end == i {
			end -= 1
		}
	}

	if end < (length + 1) {
		end += 1
	}

	return line[:end]
}

