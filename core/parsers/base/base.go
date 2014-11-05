package base

import (
	"bytes"
	"time"
	"fmt"
	"math"
)

/* Basic Parsing functionality, reused in all parsers. */
const(
	CONST_NewLine = 10
	CONST_NullChar = 0
	CONST_SpaceChar = 32
	CONST_CarriageReturn = 13
	CONST_Dot = 46
)

/* Map of byte values to numeric counterparts */
var (
	Numbers map[byte]int = map[byte]int {48: 0, 49: 1, 50: 2, 51: 3, 52: 4, 53: 5, 54: 6, 55: 7, 56: 8, 57: 9}

	/* Translation map from int to time.Month */
	IntToMonth = map[int]time.Month {
		1:  time.January,
		2:  time.February,
		3:  time.March,
		4:  time.April,
		5:  time.May,
		6:  time.June,
		7:  time.July,
		8:  time.August,
		9:  time.September,
		10: time.October,
		11: time.November,
		12: time.December,
	}

	MonthToInt = map[time.Month]int {
		time.January:	1,
		time.February:	2,
		time.March:		3,
		time.April:		4,
		time.May:		5,
		time.June:		6,
		time.July:		7,
		time.August:	8,
		time.September:	9,
		time.October:	10,
		time.November:	11,
		time.December:	12,
	}

	ERR_InvalidTimeVal = fmt.Errorf("Invalid time-val representation.")
)

/* Checks if the given argument can represent a valid number */
func ByteIsNumber (n byte) bool {
	_, ok := Numbers[n]
	return ok
}

/* Converts a slice of bytes into an int, breaking at first non numeric byte - only working for unsigned */
func ToInt (n []byte) int {
	var nr int = -1

	for _, v := range n {
		if !ByteIsNumber(v) {
			break
		} else {
			i, _ := Numbers[v]

			if nr == -1 {
				nr = i
			} else {
				nr = nr * 10 + i
			}
		}
	}

	return nr
}

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

func SplitOnSeparator(bulk []byte, sep []byte) ([][]byte) {
	return bytes.Split(bulk, sep)
}

func StringContains(raw string, needle string) bool {
	r := []byte(raw)
	n := []byte(needle)
	return bytes.Contains(r, n)
}

func ToLower(raw []byte) []byte {
	return bytes.ToLower(raw)
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

/* Parses a time-val (YYYYMMDDHHMMSS.sss - RFC-3659) representation and generates a new Time instance with obtained data */
func ParseTimeVal(timeVal []byte) (t *time.Time, err error) {
	var year, month, day, hour, min, sec, nsec int
	var inMilliseconds bool = false
	timeVal = Trim(timeVal)

	for i, c := range timeVal {
		if c != CONST_Dot {
			if !ByteIsNumber(c) {
				return t, ERR_InvalidTimeVal
			}

			d := ToInt([]byte{c})

			if i < 4 {
				/* Year part */
				year = year*10 + d
			} else if i < 6 {
				month = month*10 + d
			} else if i < 8 {
				day = day*10 + d
			} else if i < 10 {
				hour = hour*10 + d
			} else if i < 12 {
				min = min*10 + d
			} else if i < 14 {
				sec = sec*10 + d
			} else if inMilliseconds {
				nsec = nsec*10 + d
			}
		} else {
			/* Milliseconds start here */
			inMilliseconds = true
		}
	}

//	Debug point
//	fmt.Println(fmt.Sprintf("Input: %s. Output: %d/%d/%d %d:%d:%d::%d", string(timeVal), year, month, day, hour, min, sec, nsec))

	/* Check for invalid month formats */
	if _, ok := IntToMonth[month]; !ok {
		return t, ERR_InvalidTimeVal
	}

	location, err := time.LoadLocation("Etc/GMT")
	aux := time.Date(year, IntToMonth[month], day, hour, min, sec, nsec, location)
	return &aux, err
}

/* Round function courtesy of: https://gist.github.com/DavidVaini */
func round(val float64, roundOn float64, places int ) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}

/* Round function, will round to the higher float value with specified precision */
func Round(val interface{}, places int) float64 {
	var v float64
	switch val.(type) {
	case int:
		v = float64(val.(int))
	case float32:
		v = float64(val.(float32))
	case float64:
		v = val.(float64)
	}
	return  round(v, .5, places)
}

/* Generates a new TimeVal (ex: 20141030191749) from the specified Time */
func ToTimeVal(t time.Time) string {
	return fmt.Sprintf("%4d%2d%2d%2d%2d%2d", t.Year(), MonthToInt[t.Month()], t.Day(), t.Hour(), t.Minute(), t.Second())
}
