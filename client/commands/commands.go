package commands

import (
	"fmt"
	Address "github.com/ghepesdoru/bookwormFTP/core/addr"
	Command "github.com/ghepesdoru/bookwormFTP/client/command"
	BaseParser "github.com/ghepesdoru/bookwormFTP/core/parsers/base"
	FeaturesParser "github.com/ghepesdoru/bookwormFTP/core/parsers/features"
	ResourceParser "github.com/ghepesdoru/bookwormFTP/core/parsers/resource"
	Requester "github.com/ghepesdoru/bookwormFTP/client/requester"
	Status "github.com/ghepesdoru/bookwormFTP/core/codes"
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
	ERR_InvalidMode    = fmt.Errorf("Invalid transfer mode. Please consider using one of the available transfer modes (S, B, C).")
	ERR_NoPWDResult    = fmt.Errorf("Could not determine the current working directory.")
	ERR_InvalidStruct  = fmt.Errorf("Invalid file structure type. Please consider using one of the default types: F, R, P.")
	ERR_InvalidFileName= fmt.Errorf("Invalid file name.")
)

var (
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
	ok, err, msg := c.controlCommandByte(name, param, expected...)
	return ok, err, string(msg)
}

/* Implementation of all commands that require a server response message on the control connection without converting the result to string */
func (c *Commands) controlCommandByte(name string, param string, expected ...int) (bool, error, []byte) {
	if ok, err := c.IsReady(); !ok {
		return ok, err, []byte{}
	}

	command := Command.NewCommand(name, param, expected)
	c.requester.Request(command)

	return command.Success(), command.LastError(), command.Response().ByteMessage()
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

func (c *Commands) FEAT() (features *FeaturesParser.Features, err error) {
	var response []byte
	_, err, response = c.controlCommandByte("feat", EmptyString, Status.SystemStatus)
	features = FeaturesParser.FromFeaturesList(response)

	if !features.HasFeatures() {
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
	var result []byte
	_, err, result = c.controlCommandByte("mdtm", path, Status.FileStatus)

	if err == nil {
		t, err = BaseParser.ParseTimeVal(result)
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
	data, err := c.simpleDataCommand("mlsd", dir, Status.DataConnectionClose)
	res, err := ResourceParser.FromMLSxList(data)
	fmt.Println(res)

	return data, err
}

func (c *Commands) MLST(file string) ([]byte, error) {
	if len(file) == 0 {
		return []byte{}, ERR_InvalidFileName
	}

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
	return BaseParser.ToInt([]byte(response)), err
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
