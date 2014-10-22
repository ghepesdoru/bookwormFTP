package commands

import (
	"fmt"
	Command "github.com/ghepesdoru/bookwormFTP/client/command"
	Address "github.com/ghepesdoru/bookwormFTP/core/addr"
	Requester "github.com/ghepesdoru/bookwormFTP/client/requester"
	Status "github.com/ghepesdoru/bookwormFTP/core/codes"
	"strconv"
	"strings"
	"time"
)

const (
	EmptyString = ""

	/* Types */
	TYPE_Ascii     = "A"
	TYPE_Ebcdic    = "E"
	TYPE_Image     = "I"
	TYPE_LocalByte = "L"

	/* Format controls */
	FMTCTRL_NonPrint = "N"
	FMTCTRL_Telnet   = "T"
	FMTCTRL_Carriage = "C"

	/* Transfer modes */
	TRANSFER_Stream      = "S"
	TRANSFER_Block       = "B"
	TRANSFER_Compressed  = "C"
	TRANSFER_Unspecified = "U"

	FILESTRUCT_File   = "F"
	FILESTRUCT_Record = "R"
	FILESTRUCT_Page   = "P"
)

/* Default errors definition */
var (
	ERR_NotConnected   = fmt.Errorf("Invalid requester. The specified requester has no active connection to server.")
	ERR_NotReady       = fmt.Errorf("Invalid requester. The specified requester did not received a Ready message from the server.")
	ERR_NoRequester    = fmt.Errorf("Unable to execute specified action due to unnavailability of a valid Requester.")
	ERR_NoFeatures     = fmt.Errorf("Server supported features unavailable.")
	ERR_InvalidTimeVal = fmt.Errorf("Invalid time-val representation.")
	ERR_InvalidMode    = fmt.Errorf("Invalid transfer mode. Please consider using one of the available transfer modes (S, B, C).")
	ERR_NoPWDResult    = fmt.Errorf("Could not determine the current working directory.")
	ERR_InvalidStruct  = fmt.Errorf("Invalid file structure type. Please consider using one of the default types: F, R, P.")
)

var (
	/* Translation map from int to time.Month */
	IntToMonth = map[int]time.Month{
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

	/* Definition of valid representation types */
	RepresentationTypes = map[string]map[string]bool{
		/* ASCII type */
		"A": map[string]bool{
			"N": true, /* Non-Print */
			"T": true, /* Telnet format effectors */
			"C": true, /* Carriage Control (ASA) */
		},
		/* EBCDIC type */
		"E": map[string]bool{
			"N": true,
			"T": true,
			"C": true,
		},
		/* Image type */
		"I": nil,
		/* Local byte Byte size type */
		"L": nil,
	}
)

/* Define the Client Commands type */
type Commands struct {
	requester            *Requester.Requester
	hasAttachedRequester bool
}

/* Instantiates a new Commands. */
func NewCommands() (c *Commands) {
	return &Commands{nil, false}
}

/* Instantiate a new Commands instance. This can be used as a commands provider in the client */
func NewCommandsProvider(hostURL string) (c *Commands, err error) {
	var requester *Requester.Requester

	if requester, err = Requester.NewRequester(hostURL); err == nil {
		c = NewCommands()
		_, err = c.AttachRequester(requester)

		if err != nil {
			c = nil
		}
	}

	return
}

/* Wrapper around Command.NewCommand for fast instantiations */
func NewCommand(command string, parameters string, expectedStatus ...int) *Command.Command {
	return Command.NewCommand(command, parameters, expectedStatus)
}

/* Attaches the specified Requester instance to the current Commands. */
func (c *Commands) AttachRequester(requester *Requester.Requester) (ok bool, err error) {
	if !requester.IsConnected() {
		err = ERR_NotConnected
	} else if !requester.IsReady() {
		err = ERR_NotReady
	} else {
		c.requester = requester
		c.hasAttachedRequester = true
		ok = true
	}

	return
}

/* Checks if the current Commands has a valid requester */
func (c *Commands) IsReady() (ok bool, err error) {
	if c.requester == nil {
		err = ERR_NoRequester
	} else if !c.requester.IsConnected() {
		err = ERR_NotConnected
	} else if !c.requester.IsReady() {
		err = ERR_NotReady
	} else {
		ok = true
	}

	return
}

/* Requester getter */
func (c *Commands) Requester() *Requester.Requester {
	if c.hasAttachedRequester {
		return c.requester
	}

	return nil
}

/* Implementation for all commands that only have to return a status flag and eventual errors */
func (c *Commands) simpleControlCommand(name string, param string, expected ...int) (bool, error) {
	if ok, err := c.IsReady(); !ok {
		return ok, err
	}

	command := Command.NewCommand(name, param, expected)
	c.requester.Request(command)

	return command.Success(), command.LastError()
}

/* Implementation for all commands that require a server response message on the control connection */
func (c *Commands) controlCommand(name string, param string, expected ...int) (bool, error, string) {
	if ok, err := c.IsReady(); !ok {
		return ok, err, EmptyString
	}

	command := Command.NewCommand(name, param, expected)
	c.requester.Request(command)

	return command.Success(), command.LastError(), command.Response().Message()
}

/* Implementation for all commands that only require to grab small amounts of data from the data connection. */
func (c *Commands) simpleDataCommand(name string, param string, expected ...int) ([]byte, error) {
	if ok, err := c.IsReady(); !ok {
		return []byte{}, err
	}

	command, data := c.requester.RequestData(Command.NewCommand(name, param, expected))
	return data, command.LastError()
}

func asUpperNormalized(s string) string {
	return strings.TrimSpace(strings.ToUpper(s))
}

/* Parses a time-val (YYYYMMDDHHMMSS.sss - RFC-3659) representation and generates a new Time instance with obtained data */
func (c *Commands) parseTimeVal(timeVal string) (t *time.Time, err error) {
	var year, month, day, hour, min, sec, nsec int
	var dot rune = rune('.')
	var inMilliseconds bool = false
	timeVal = strings.TrimSpace(timeVal)

	for i, c := range timeVal {
		if c != dot {
			d, err := strconv.Atoi(string(c))

			if err != nil {
				/* Stop parsing on wrong formatted data */
				return t, ERR_InvalidTimeVal
			}

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

	/* Check for invalid month formats */
	if _, ok := IntToMonth[month]; !ok {
		return t, ERR_InvalidTimeVal
	}

	location, err := time.LoadLocation("Etc/GMT")
	aux := time.Date(year, IntToMonth[month], day, hour, min, sec, nsec, location)
	return &aux, err
}

/* COMMANDS definitions --------------------------------------------------------------------------- */
func (c *Commands) ABOR() (bool, error) {
	return c.simpleControlCommand("abor", EmptyString, 0)
}

func (c *Commands) ACCT(accountInfo string) (bool, error) {
	return c.simpleControlCommand("acct", accountInfo, 0)
}

func (c *Commands) ADAT() (bool, error) {
	return c.simpleControlCommand("adat", EmptyString, 0)
}

func (c *Commands) ALGS() (bool, error) {
	return c.simpleControlCommand("algs", EmptyString, 0)
}

func (c *Commands) ALLO() (bool, error) {
	return c.simpleControlCommand("allo", EmptyString, 0)
}

func (c *Commands) APPE() (bool, error) {
	return c.simpleControlCommand("appe", EmptyString, 0)
}

func (c *Commands) AUTH() (bool, error) {
	return c.simpleControlCommand("auth", EmptyString, 0)
}

func (c *Commands) AUTH_PLUS() (bool, error) {
	return c.simpleControlCommand("auth+", EmptyString, 0)
}

func (c *Commands) CCC() (bool, error) {
	return c.simpleControlCommand("ccc", EmptyString, Status.PositiveCompletion)
}

func (c *Commands) CDUP() (bool, error) {
	return c.simpleControlCommand("cdup", EmptyString, Status.FileActionOk)
}

func (c *Commands) CONF() (bool, error) {
	return c.simpleControlCommand("conf", EmptyString, 0)
}

func (c *Commands) CWD(path string) (bool, error) {
	return c.simpleControlCommand("cwd", path, Status.FileActionOk)
}

func (c *Commands) DELE(path string) (bool, error) {
	return c.simpleControlCommand("cwd", path, Status.FileActionOk)
}

func (c *Commands) ENC() (bool, error) {
	return c.simpleControlCommand("enc", EmptyString, 0)
}

func (c *Commands) EPRT(port uint) (bool, error) {
	addr := c.requester.GetHostAddr()
	addr.Port = int(port)
	return c.simpleControlCommand("eprt", addr.ToExtendedPortSpecifier(), Status.PositiveCompletion)
}

func (c *Commands) EPSV() (ok bool, err error) {
	var response string
	ok, err, response = c.controlCommand("epsv", EmptyString, Status.ExtendedPassiveMode)
	if ok {
		ok, err = c.requester.RegisterDataAddr(Address.FromExtendedPortSpecifier(response))
	}

	return
}

func (c *Commands) FEAT() (features map[string]string, err error) {
	var response string
	var parts []string

	_, err, response = c.controlCommand("feat", EmptyString, Status.SystemStatus)
	if err == nil {
		features = make(map[string]string)

		if parts = strings.Split(response, "\r\n"); len(parts) == 0 {
			parts = strings.Split(response, "\n")
		}

		length := len(parts) - 1
		for i, line := range parts {
			line = strings.TrimSpace(line)

			if i == 0 || i == length {
				/* First line, can be "Features: ", last line can be "END" */
				if l := strings.ToLower(line); l == "features:" || l == "end" {
					/* Skip the two lines in feature extraction */
					continue
				}
			}

			aux := strings.Split(line, " ")
			l := len(aux)

			if l > 0 {
				if l > 1 {
					features[strings.ToUpper(aux[0])] = strings.Join(aux[1:], " ")
				} else {
					features[strings.ToUpper(aux[0])] = EmptyString
				}
			}
		}
	}

	if len(features) == 0 {
		err = ERR_NoFeatures
	}

	return features, err
}

func (c *Commands) HELP(with string) (string, error) {
	_, err, response := c.controlCommand("help", with, Status.HelpMessage)
	return string(response), err
}

func (c *Commands) HOST(virtualHost string) (bool, error) {
	return c.simpleControlCommand("host", virtualHost, Status.PositiveCompletion)
}

func (c *Commands) LANG(lang string) (bool, error) {
	return c.simpleControlCommand("lang", lang, Status.PositiveCompletion)
}

func (c *Commands) LIST(path string) ([]byte, error) {
	return c.simpleDataCommand("list", path, Status.DataConnectionClose)
}

func (c *Commands) MDTM(path string) (t *time.Time, err error) {
	var result string
	_, err, result = c.controlCommand("mdtm", path, Status.FileStatus)

	if err == nil {
		t, err = c.parseTimeVal(result)
	}

	return
}

func (c *Commands) MIC() (bool, error) {
	return c.simpleControlCommand("mic", EmptyString, 0)
}

func (c *Commands) MKD(dir string) (bool, error) {
	return c.simpleControlCommand("mkd", dir, Status.Pathname)
}

func (c *Commands) MLSD(dir string) ([]byte, error) {
	return c.simpleDataCommand("mlsd", dir, Status.DataConnectionClose)
}

func (c *Commands) MLST(file string) ([]byte, error) {
	return c.simpleDataCommand("mlst", file, Status.FileActionOk)
}

func (c *Commands) MODE(mode string) (bool, error) {
	mode = asUpperNormalized(mode)

	if mode != TRANSFER_Block && mode != TRANSFER_Compressed && mode != TRANSFER_Stream {
		return false, ERR_InvalidMode
	}

	return c.simpleControlCommand("mode", mode, Status.PositiveCompletion)
}

func (c *Commands) NLST(path string) ([]byte, error) {
	return c.simpleDataCommand("nlst", path, Status.DataConnectionClose)
}

func (c *Commands) NOOP() (bool, error) {
	return c.simpleControlCommand("noop", EmptyString, Status.PositiveCompletion)
}

func (c *Commands) OPTS(command string, option string) (bool, error) {
	return c.simpleControlCommand("opts", command+" "+option, Status.PositiveCompletion)
}

func (c *Commands) PASS(password string) (bool, error) {
	return c.simpleControlCommand("pass", password, Status.UserLoggedIn)
}

func (c *Commands) PASV() (ok bool, err error) {
	ok, err, response := c.controlCommand("pasv", EmptyString, Status.PassiveMode)
	if ok {
		ok, err = c.requester.RegisterDataAddr(Address.FromPortSpecifier(response))
	}

	return
}

func (c *Commands) PBSZ() (bool, error) {
	return c.simpleControlCommand("pbsz", EmptyString, 0)
}

func (c *Commands) PBSZ_PLUS() (bool, error) {
	return c.simpleControlCommand("pbsz+", EmptyString, 0)
}

func (c *Commands) PORT(port uint) (bool, error) {
	addr := c.requester.GetHostAddr()
	addr.Port = int(port)
	return c.simpleControlCommand("port", addr.ToPortSpecifier(), Status.PositiveCompletion)
}

func (c *Commands) PROT() (bool, error) {
	return c.simpleControlCommand("prot", EmptyString, 0)
}

func (c *Commands) PROT_PLUS() (bool, error) {
	return c.simpleControlCommand("prot+", EmptyString, 0)
}

func (c *Commands) PWD() (dir string, err error) {
	var start, end int = -1, -1
	var sep rune = '"'

	_, err, dir = c.controlCommand("pwd", EmptyString, Status.Pathname)

	for i, c := range dir {
		if c == sep {
			if start == -1 {
				start = i
			} else {
				end = i
				break
			}
		}
	}

	if start > -1 && end > -1 {
		dir = dir[start+1 : end]
	} else {
		dir = EmptyString
	}

	return
}

func (c *Commands) QUIT() (exitMessage string, err error) {
	_, err, exitMessage = c.controlCommand("quit", EmptyString, Status.ClosingControlConnection)
	return exitMessage, err
}

func (c *Commands) REIN() (bool, error) {
	return c.simpleControlCommand("rein", EmptyString, Status.Ready)
}

func (c *Commands) REST(marker string) (bool, error) {
	return c.simpleControlCommand("rest", marker, Status.FileActionPending)
}

func (c *Commands) REST_PLUS(marker string) (bool, error) {
	return c.simpleControlCommand("rest+", marker, Status.FileActionPending)
}

func (c *Commands) RETR(path string) (bool, error) {
	return false, nil
}

func (c *Commands) RMD(path string) (bool, error) {
	return c.simpleControlCommand("rmd", path, Status.FileActionOk)
}

func (c *Commands) RNFR(fileName string) (bool, error) {
	return c.simpleControlCommand("rnfr", fileName, Status.FileActionPending)
}

func (c *Commands) RNTO(fileName string) (bool, error) {
	return c.simpleControlCommand("rnto", fileName, Status.FileActionOk)
}

func (c *Commands) SITE(params string) (bool, error) {
	return c.simpleControlCommand("site", params, Status.PositiveCompletion)
}

func (c *Commands) SIZE(fileName string) (int, error) {
	_, err, response := c.controlCommand("size", fileName, Status.FileStatus)
	return Status.ToInt([]byte(response)), err
}

func (c *Commands) SMNT(path string) (bool, error) {
	return c.simpleControlCommand("smnt", path, Status.FileActionOk)
}

func (c *Commands) STAT(resource string) (string, error) {
	_, err, response := c.controlCommand("stat", resource, Status.SystemStatus, Status.FileStatus)
	return string(response), err
}

func (c *Commands) STOR() {

}

func (c *Commands) STOU() {

}

func (c *Commands) STRU(structure string) (bool, error) {
	structure = asUpperNormalized(structure)
	if structure != FILESTRUCT_File && structure != FILESTRUCT_Page && structure != FILESTRUCT_Record {
		return false, ERR_InvalidStruct
	}

	return c.simpleControlCommand("stru", structure, Status.PositiveCompletion)
}

func (c *Commands) SYST() (string, error) {
	_, err, sysType := c.controlCommand("syst", EmptyString, Status.NAMEType)
	return string(sysType), err
}

func (c *Commands) TYPE(reprType string) (bool, error) {
	return c.simpleControlCommand("type", reprType, Status.PositiveCompletion)
}

func (c *Commands) USER(username string) (bool, error) {
	return c.simpleControlCommand("user", username, Status.UserNameOk)
}
