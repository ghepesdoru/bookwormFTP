package client

import (
	"fmt"
	"net"
	"time"
	"path"
	"strconv"
	"net/url"
	"strings"
	Address "github.com/ghepesdoru/bookwormFTP/core/addr"
	Command "github.com/ghepesdoru/bookwormFTP/client/command"
	Commands "github.com/ghepesdoru/bookwormFTP/core/commands"
	Status "github.com/ghepesdoru/bookwormFTP/core/codes"
	Reader "github.com/ghepesdoru/bookwormFTP/core/reader"
	Response "github.com/ghepesdoru/bookwormFTP/core/response"
	Parser "github.com/ghepesdoru/bookwormFTP/core/parser"
	Settings "github.com/ghepesdoru/bookwormFTP/client/settings"
	Credentials "github.com/ghepesdoru/bookwormFTP/core/credentials"
)

/* Constants definition */
const (
	CONST_ClientNetwork = "tcp"
	CONST_ServerPort 	= 21
	CONST_DefaultUser 	= "anonymous"
	CONST_DefaultPass	= ""
	CONST_EmptyString	= ""
	CONST_CommandRetries= 3
	CONST_DataPort	 	= -1
	CONST_Comma			= ","

	/* Connection option names */
	OPT_Connected 		= "connected"
	OPT_ServerReady 	= "ready"
	OPT_InitialPath 	= "initial_path"
	OPT_Disconnected 	= "disconnected"
	OPT_Authenticated 	= "logged_in"
	OPT_DataPort		= "client_data_port"
	OPT_PassiveMode		= "passive"
	OPT_ExtendedPassive = "extended_passive"
	OPT_CurrentDir		= "cwd"
	OPT_Account			= "account"
	OPT_AccountEnabled = "account_active"
	OPT_TransferMode	= "transfer_mode"
	OPT_DataType		= "data_type"
	OPT_FormatControl	= "format_control"
	OPT_ByteSize		= "byte_size"
	OPT_FileStructure	= "file_structure"

	/* Types */
	TYPE_Ascii			= "A"
	TYPE_Ebcdic			= "E"
	TYPE_Image			= "I"
	TYPE_LocalByte		= "L"

	/* Format controls */
	FMTCTRL_NonPrint	= "N"
	FMTCTRL_Telnet		= "T"
	FMTCTRL_Carriage	= "C"

	/* Transfer modes */
	TRANSFER_Stream		= "S"
	TRANSFER_Block		= "B"
	TRANSFER_Compressed	= "C"
	TRANSFER_Unspecified= "U"

	FILESTRUCT_File		= "F"
	FILESTRUCT_Record	= "R"
	FILESTRUCT_Page		= "P"
)

/* Default errors definition */
var (
	ERR_InvalidClientAddress = fmt.Errorf("Invalid client address.")
	ERR_ResponseParsingError = fmt.Errorf("An error triggered while parsing the server response.")
	ERR_UnconsumedResponses	 = fmt.Errorf("Acumulation of unconsummed responses from the server.")
	ERR_NoServerResponse	 = fmt.Errorf("Unable to fetch a response from server at this time.")
	ERR_RestartSequence		 = fmt.Errorf("Restart sequence.")
	ERR_ServerNotReady		 = fmt.Errorf("Server is disconnected or otherwise unavailable.")
	ERR_ReinNotImplemented	 = fmt.Errorf("Server state reinitialization not supported. (REIN)")
	ERR_NoServerFeatures	 = fmt.Errorf("Server supported features unavailable.")
	ERR_NoPWDResult			 = fmt.Errorf("Could not determine the current working directory path.")
	ERR_InvalidListCommand	 = fmt.Errorf("Unable to list requested content. Please consider putting the client in passive mode or providing a client port.")
	ERR_InvalidIpAndPortRepr = fmt.Errorf("Invalid ip and port representation. Expected ip8bit,ip8bit,ip8bit,ip8bit,port8bit,port8bit")
	ERR_InvalidDataConn		 = fmt.Errorf("Unable to establish a data link with the remote server.")
	ERR_InvalidTimeVal		 = fmt.Errorf("Invalid time-val representation.")
	ERR_InvalidMKDPath		 = fmt.Errorf("Invalid path for directory creation. An error took place while recursively generating the path components.")
	ERR_LoginAccountRequired = fmt.Errorf("Please specify an account and restart the authentication sequence.")
	ERR_InvalidType			 = fmt.Errorf("Invalid type specified. Please consider using one of the available types (A, E, I, L).")
	ERR_InvalidFMTCTRL		 = fmt.Errorf("Invalid format control. Please consider using one of the avialable format controls (N, T, C).")
	ERR_InvalidByteSize		 = fmt.Errorf("Invalid byte size for Local byte Byte size type.")
	ERR_InvalidTransferMode	 = fmt.Errorf("Invalid transfer mode. Please consider using one of the available transfer modes (S, B, C).")
	ERR_InvalidPort			 = fmt.Errorf("Invalid data connection port. Consider a port number higher then 30 000.")
	ERR_InvalidFileStructure = fmt.Errorf("Invalid file structure type. Please consider using one of the default types: F, R, P.")
	ERR_SelectVirtualHostBeforeAuth = fmt.Errorf("The current connection can not be reinititialized. Please start a new connection and chose the virtual server before the authentication process.")

	/* Error formats */
	ERRF_InvalidCommandName = "Command error: Unrecognized command %s."
	ERRF_InvalidCompletionStatus = "Command error: %s completed without meeting any of the %s status. Completion status: %d, completion message %s"
	ERRF_InvalidCommandOutOfSequence = "Command error: %s could not complete. Use a sequence for fequential commands. Intermediary status: %d, message: %s"
	ERRF_CommandMaxRetries = "Command error: %s reached the maximum number of retries. Transient Negative Completion reply status %d, message: %s"
	ERRF_CommandFailure = "Command failure: %d %s"
	ERRF_MissingPortInHost = "missing port in address"
)

/* Other global declarations */
var (
	/* Translation map from int to time.Month */
	IntToMonth 			= map[int]time.Month {
		1: 	time.January,
		2: 	time.February,
		3: 	time.March,
		4: 	time.April,
		5: 	time.May,
		6: 	time.June,
		7: 	time.July,
		8: 	time.August,
		9: 	time.September,
		10:	time.October,
		11:	time.November,
		12:	time.December,
	}

	/* Definition of valid representation types */
	RepresentationTypes = map[string]map[string]bool {
		/* ASCII type */
		"A": map[string]bool {
			"N": true,	/* Non-Print */
			"T": true,	/* Telnet format effectors */
			"C": true,	/* Carriage Control (ASA) */
		},
		/* EBCDIC type */
		"E": map[string]bool {
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

/* BookwormFTP Client type definition */
type Client struct {
	connection net.Conn
	dataAddr *net.TCPAddr
	reader *Reader.Reader
	credentials *Credentials.Credentials
	settings *Settings.Settings
}

/* Bookmark FTP Client builder */
func NewClient(address string) (client *Client, err error) {
	var urlData *url.URL
	var host string
	var credentials *Credentials.Credentials
	var conn net.Conn
	var authenticate, ok bool = true, false
	var settings *Settings.Settings = Settings.NewSettings(
		/* Define current connection settings with default values */
		Settings.NewOption(OPT_Connected, false),			/* There is a connection to the host */
		Settings.NewOption(OPT_ServerReady, false),			/* The server send it's welcome message? */
		Settings.NewOption(OPT_Disconnected, false),		/* A QUIT command was called? */
		Settings.NewOption(OPT_Authenticated, false),		/* A user is currently authenticated */
		Settings.NewOption(OPT_PassiveMode, false), 		/* Client is not in passive mode at connection time */
		Settings.NewOption(OPT_ExtendedPassive, false), 	/* Extended passive mode */
		Settings.NewOption(OPT_DataPort, CONST_DataPort), 	/* Register a default invalid data port */
		Settings.NewOption(OPT_Account, CONST_EmptyString),	/* Default account */
		Settings.NewOption(OPT_AccountEnabled, false),		/* No active account */
		Settings.NewOption(OPT_TransferMode, TRANSFER_Unspecified), /* The connection has no specified transfer mode at this point */
		Settings.NewOption(OPT_DataType, TYPE_Ascii), 		/* Presume ASCII as default data type */
		Settings.NewOption(OPT_FormatControl, FMTCTRL_NonPrint), /* Presume non print format control */
		Settings.NewOption(OPT_ByteSize, 8),				/* Asume a 8 bit byte size */
		Settings.NewOption(OPT_FileStructure, FILESTRUCT_File), /* Default to the file structure of file */
		Settings.NewOption(OPT_CurrentDir, "/"),			/* Defines the default current working directory as / */
	)

	/* Extract the url parts */
	urlData, err = url.Parse(address)

	if err != nil {
		err = ERR_InvalidClientAddress
		return
	}

	if !strings.Contains(urlData.Host, ":") {
		host = fmt.Sprintf("%s:%d", urlData.Host, CONST_ServerPort)
	} else {
		host = urlData.Host
	}

	/* Check if any credentials are passed in the url */
	if urlData.User != nil {
		if password, ok := urlData.User.Password(); ok {
			credentials, err = Credentials.NewCredentials(urlData.User.Username(), password)
		} else {
			credentials, err = Credentials.NewCredentials(urlData.User.Username(), CONST_DefaultPass)
		}
	}

	/* Use anonymous login as default for cases where credentials are not provided or otherwise invalid */
	if credentials == nil || err == Credentials.ERR_UsernameToShort {
		/* Create anonymous credentials */
		credentials, _ = Credentials.NewCredentials(CONST_DefaultUser, CONST_DefaultPass)

		/* No custom credentials whare delivered. Do not authenticate at this time */
		authenticate = false
	}

	/* Remember the initially requested path */
	settings.Get(OPT_InitialPath).Set(urlData.Path)

	/* Connect to the remote host */
	conn, err = net.Dial(CONST_ClientNetwork, host)
	if err != nil {
		return /* Return with the original Dial generated error */
	}

	/* Connected successfully */
	settings.Get(OPT_Connected).Set(true)

	/* Instantiate the new client */
	client = &Client{conn, nil, Reader.NewReader(conn), credentials, settings}

	/* Grab server greeting, and check for server ready status */
	welcomeMessage, _ := client.getResponse()
	if welcomeMessage != nil {
		if Status.Ready == welcomeMessage.Status() {
			/* Server ready */
			settings.Get(OPT_ServerReady).Set(true)
		}
	}

	/* Authenticate with the provided user and password if the server address contained a user and password */
	if authenticate {
		ok, err = client.Authenticate(client.credentials)

		if ok {
			/* Authenticated. Navigate to the specified path (if any) */
			if urlData.Path != CONST_EmptyString {
				_, err = client.ChangeDirectory(urlData.Path)
				client.settings.Add(OPT_InitialPath, CONST_EmptyString)
			}
		}
	} else {
		/* Manual authentication at a later time. Remember the specified path (if any) */
		if urlData.Path != CONST_EmptyString {
			client.settings.Add(OPT_InitialPath, CONST_EmptyString).Set(urlData.Path)
		}
	}

	return
}

/* Wrapper around Command.NewCommand for fast instantiations */
func NewCommand(command string, parameters string, expectedStatus ...int) *Command.Command {
	return Command.NewCommand(command, parameters, expectedStatus)
}

/* Flushes the stack of server send messages at time of call, uses the response parser to generate Response instances */
func (c *Client) getResponse() (response *Response.Response, err error) {
	var parser *Parser.Parser = Parser.NewParser()
	var raw []byte = c.reader.Get()

	/* Parse the read content */
	parser.ParseBlock(raw)

	if parser.HasErrors() {
		/* Debug point */
		for err := range parser.Errors() {
			fmt.Println("Parsing error: ", err)
		}

		err = ERR_ResponseParsingError
	} else {
		if parser.Length() > 1 {
			/* Debug point. This will happen if multiple commands are executed before calling getResponse */
			fmt.Println("Accumulation of responses not consumed: ", string(raw))

			err = ERR_UnconsumedResponses
		} else {
			response = parser.Get()
		}
	}

	return
}

/* Data response fetcher */
func (c *Client) getDataResponse() (response string, err error) {
	conn, err := net.DialTCP(CONST_ClientNetwork, nil, c.dataAddr)

	if err == nil {
		reader := Reader.NewReader(conn)
		data := reader.Get()
		response = string(data)
		reader.StopReading()
		conn.Close()
	}

	return
}

/* Checks the server availability for command execution */
func (c *Client) serverReady() (ok bool) {
	return (
		c.settings.Get(OPT_Connected).Is(true) &&
		c.settings.Get(OPT_ServerReady).Is(true) &&
		c.settings.Get(OPT_Disconnected).Is(false))
}

/* Makes requests to the server based on provided Command contents */
func (c *Client) request(command *Command.Command) (bool, error) {
	var toSend []byte
	var EOL []byte = []byte("\r\n")

	if command.HasParameters() {
		toSend = []byte(command.Name() + " " + command.Parameters())
	} else {
		toSend = []byte(command.Name())
	}

	/* Send the request to the server */
	n, err := c.connection.Write(append(toSend, EOL...))

	return n > 0, err
}

/* Executes a command (wrapper around request, takes care of response reading, error handling, and is status aware) */
func (c *Client) execute(command *Command.Command, isSequence bool, execute bool, leftRetries int) {
	/* Initialize variables to defaults */
	var err error

	if command.Name() == Commands.CONST_UnknownCommand {
		/* Invalid command name */
		command.AddError(fmt.Errorf(ERRF_InvalidCommandName, command.Name()))
		return
	}

	if !c.serverReady() {
		/* Do not make requests on closed connections */
		command.AddError(ERR_ServerNotReady)
		return
	}

	/* Execute the command */
	if execute {
		_, err = c.request(command)

		/* Error communicating to server */
		if err != nil {
			command.AddError(err)
			return
		}
	}

	if command.Name() == "LIST" && execute {
		go func () {
			fmt.Println("Special data requiring method:")
			fmt.Println(c.getDataResponse())
		}()

		c.execute(command, isSequence, false, leftRetries)
		return
 	}

	/* Get the server response */
	command.AttachResponse(c.getResponse())

	if command.Response() == nil {
		/* Empty server response */
		command.AddError(ERR_NoServerResponse)

		/* Attach an empty response to the command to ensure interface chaining capabilities. */
		command.AttachResponse(&Response.Response{}, nil)

		return
	}

	/* Check response status to determine the execution completion */
	status := command.Response().Status()

	/* Check relay status for next action */
	first := status / 100
//	second := (status / 10) % 10
	// TODO: Take the second digit into account in reply management also?

	if first == 1 {
		/* Positive Preliminary reply - wait for a new response */
		c.execute(command, isSequence, false, leftRetries)
		return
	} else if first == 2 {
		/* Positive Completion reply - action completed successfully, no matter of the expected status */
		if !command.IsExpectedStatus(status) {
			command.AddError(fmt.Errorf(ERRF_InvalidCompletionStatus, command.Name(), command.ExpectedStatus(), status, command.Response().Message()))
		}
	} else if first == 3 {
		/* Positive Intermediate reply - sequence of commands mandatory */
		if !isSequence {
			/* Error: Invalid single command. Use a sequence */
			command.AddError(fmt.Errorf(ERRF_InvalidCommandOutOfSequence, command.Name(), status, command.Response().Message()))
		}
	} else if first == 4 {
		if leftRetries == 0 {
			/* Stop the retry process. The acction failed to many times. */
			command.AddError(fmt.Errorf(ERRF_CommandMaxRetries, command.Name(), status, command.Response().Message()))
		} else {
			/* Transient Negative Completion reply - repeat the command(s) */
			if (isSequence) {
				/* Reset the sequence, this is a temporary error */
				command.AddError(ERR_RestartSequence)
			} else {
				/* Try again to execute this command */
				c.execute(command, isSequence, true, leftRetries - 1)
				return
			}
		}
	} else if first == 5 {
		/* Permanent Negative Completion reply - failure. Forward the server error message */
		command.AddError(fmt.Errorf(ERRF_CommandFailure, status, command.Response().Message()))
	}

	if command.Success() {
		/* Command completed as expected - Debug point */
		fmt.Println("Successfull command: ", command.Name(), command.Response())
	} else {
		/* Command failed - Debug point */
		fmt.Println("Error executing: ", command.Name(), command.LastError())
	}

	return
}

/* Executes a specified sequence of commands */
func (c *Client) sequence(commands []*Command.Command) (ok bool, last *Command.Command) {
	var leftRetries int = CONST_CommandRetries
	var retry bool = false

	for leftRetries > 0 {
		for _, command := range commands {
			last = command
			c.execute(command, true, true, leftRetries)

			/* Take into consideration sequence retries */
			if command.LastError() == ERR_RestartSequence {
				leftRetries -= 1
				retry = true
				command.FlushErrors()
				break
			} else if command.Success() != true {
				break
			}
		}

		if retry != true {
			break
		}
	}

	return last.Success(), last
}

/* Checks if the client is in any of the passive modes */
func (c *Client) inPassiveMode() bool {
	if c.settings.Get(OPT_PassiveMode).Is(true) || c.settings.Get(OPT_ExtendedPassive).Is(true) {
		return true
	}

	return false
}

/* Parses a time-val (YYYYMMDDHHMMSS.sss - RFC-3659) representation and generates a new Time instance with obtained data */
func (c *Client) parseTimeVal(timeVal string) (t *time.Time, err error) {
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
				year = year * 10 + d
			} else if i < 6 {
				month = month * 10 + d
			} else if  i < 8 {
				day = day * 10 + d
			} else if i < 10 {
				hour = hour * 10 + d
			} else if i < 12 {
				min = min * 10 + d
			} else if i < 14 {
				sec = sec * 10 + d
			} else if inMilliseconds {
				nsec = nsec * 10 + d
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

/* Extracts the current path parts (directories and file) from the specified input */
func (c *Client) extractPathElements(p string) (dir string, file string) {
	p = path.Clean(p)
	return path.Split(p)
}

/* Given a relative path, will concatenate it with the current working directory and normalize it */
func (c *Client) toAbsolutePath (d string) string {
	if !path.IsAbs(d) {
		/* Not an absolute path, concatenate the relative path to the working directory, and normalize them together  */
		d = c.settings.Get(OPT_CurrentDir).ToString() + "/" + d
		d, _ = c.extractPathElements(d)
	}

	return d
}

/* Make a request to the server */
func (c *Client) Request(command *Command.Command) (*Command.Command) {
	c.execute(command, false, true, CONST_CommandRetries)
	return command
}

/* Make a sequence of requests */
func (c *Client) Sequence(commands ...*Command.Command) (bool, *Command.Command) {
	return c.sequence(commands)
}

/* COMMANDS Implementations ----------------------------------------------------------------------------------------- */

// TODO: AbortFileTransfer

/* Specify the account in use to the server */
func (c *Client) Account(accountInfo string) (bool, error) {
	if c.settings.Get(OPT_AccountEnabled).Is(true) {
		if c.settings.Get(OPT_Account).Is(accountInfo) {
			/* Same account, return */
			return true, nil
		} else {
			/* Reinitialize connection */
			if ok, _ := c.Reinitialize(); !ok {
				/* Connection reinitialization not supported */
				return false, ERR_ReinNotImplemented
			} else {
				/* Connection reinitialized. Reset the account enable flag and call Account again */
				c.settings.Get(OPT_AccountEnabled).Reset()
				return c.Account(accountInfo)
			}
		}
	}

	command := c.Request(NewCommand("acct", accountInfo, 0))

	if command.Success() {
		c.settings.Get(OPT_Account).Set(accountInfo)
		c.settings.Get(OPT_AccountEnabled).Set(accountInfo)
	}

	return command.Success(), command.LastError()
}

/* Authenticate the user with provided credentials */
func (c *Client) Authenticate(credentials *Credentials.Credentials) (bool, error) {
	var modified bool = false
	var command *Command.Command = &Command.Command{}

	if credentials == nil {
		/* Fallback on existing credentials */
		credentials = c.credentials
	}

	if credentials.Username() != c.credentials.Username() || credentials.Password() != c.credentials.Password() {
		/* Username or password modified, update the credentials */
		modified = true
		c.credentials = credentials
	}

	/* Authenticate the user on first try */
	if c.settings.Get(OPT_Authenticated).Is(false) {
		_, command = c.Sequence(
			NewCommand("user", credentials.Username(), Status.UserNameOk),
			NewCommand("pass", credentials.Password(), Status.UserLoggedIn, Status.AccountForLogin),
		)
	} else if modified {
		/* Reset the connection if supported (REIN) */
		ok, _ := c.Reinitialize()

		if ok {
			/* Call Authenticate again after resetting the flag */
			c.settings.Get(OPT_Authenticated).Reset()
			return c.Authenticate(credentials)
		} else {
			/* Unable to reinitialize connection. */
			command.AddError(ERR_ReinNotImplemented)
		}
	}

	if command.Response().Status() == Status.AccountForLogin {
		/* An account is required to continue */
		if c.settings.Get(OPT_Account).ToString() != CONST_EmptyString {
			if _, err := c.Account(c.settings.Get(OPT_Account).ToString()); err != nil {
				command.AddError(err)
			}
		} else {
			/* Set an account and restart the sequence */
			command.AddError(ERR_LoginAccountRequired)
		}
	}

	if command.Success() {
		/* Notify user authenticated */
		c.settings.Get(OPT_Authenticated).Set(true)

		/* Check if there is any initial path to navigate to */
		if !c.settings.Get(OPT_InitialPath).Is(CONST_EmptyString) {
			/* Ignore errors, this is not an authentication problem */
			_, _ = c.ChangeDirectory(c.settings.Get(OPT_InitialPath).ToString())

			/* Reset the initial path to it's default value (empty string) */
			c.settings.Get(OPT_InitialPath).Reset()
		}
	}

	return command.Success(), command.LastError()
}

// TODO: func (c *Client) AllocateDiskSpace() {}
// TODO: func (c *Client) Append() {}

/* Allows specification of an extended address for the data connection */
func (c *Client) SpecifyExtendedAddress (port int) (bool, error) {
	if port < 1 {
		/* Invalid port number */
		return false, ERR_InvalidPort
	}

	address := Address.FromConnection(c.connection)
	address.Port = port

	command := c.Request(NewCommand("eprt", address.ToExtendedPortSpecifier(), Status.PositiveCompletion))

	return command.Success(), command.LastError()
}

/* Gives the ability to specify a non default data port for the data connection */
func (c *Client) SpecifyPort (port int) (bool, error) {
	if port < 1 {
		/* Invalid port number */
		return false, ERR_InvalidPort
	}

	addr := Address.FromConnection(c.connection)
	addr.Port = port
	command := c.Request(NewCommand("port", addr.ToPortSpecifier(), Status.PositiveCompletion))

	return command.Success(), command.LastError()
}

/* Changes the current directory on the server to the specified one - supports paths relative to the currently selected directory */
func (c *Client) ChangeDirectory(p string) (bool, error) {
	/* Normalize the path to an absolute path */
	dir := c.toAbsolutePath(p)

	/* Do not request changes to the same current directory */
	if p == c.settings.Get(OPT_CurrentDir).ToString() {
		return true, nil
	}

	command := c.Request(NewCommand("cwd", dir, Status.FileActionOk))

	if command.Success() {
		/* Remember the new path */
		c.settings.Get(OPT_CurrentDir).Set(p)
	}

	return command.Success(), command.LastError()
}

/* Delete the specified file on the remote server */
func (c *Client) DeleteFile(fileName string) (ok bool, err error) {
	command := c.Request(NewCommand("dele", fileName, Status.FileActionOk))
	return command.Success(), command.LastError()
}

/* Disconnect Command functionality */
func (c *Client) Disconnect() (quitMessage string, err error) {
	command := c.Request(NewCommand("quit", CONST_EmptyString, Status.ClosingControlConnection))

	if command.Success() {
		/* Notify the server disconnection */
		c.settings.Get(OPT_Disconnected).Set(true)
		c.settings.Get(OPT_ServerReady).Reset()
		c.settings.Get(OPT_Authenticated).Reset()

		/* Close the TCP connection */
		c.connection.Close()
		c.settings.Get(OPT_Connected).Reset()
	}

	return command.Response().Message(), command.LastError()
}

/* Remove the specified directory */
func (c *Client) DeleteDirectory(dirName string) (bool, error) {
	var dir string
	dir = c.toAbsolutePath(dirName)

	command := c.Request(NewCommand("rmd", dir, Status.FileActionOk))
	return command.Success(), command.LastError()
}

/* Server supported features getter */
func (c *Client) Features() (map[string]string, error) {
	var raw string
	var parts []string
	var length int
	var features map[string]string

	command := c.Request(NewCommand("feat", CONST_EmptyString, Status.SystemStatus))

	if command.Success() {
		raw = command.Response().Message()
		features = make(map[string]string)

		if parts = strings.Split(raw, "\r\n"); len(parts) == 0 {
			parts = strings.Split(raw, "\n")
		}

		length = len(parts) - 1
		for i, line := range(parts) {
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
					features[strings.ToUpper(aux[0])] = CONST_EmptyString
				}
			}
		}
	}

	if len(features) == 0 {
		command.AddError(ERR_NoServerFeatures)
	}

	return features, command.LastError()
}

/* Asks the server for the date and time of the last file modification */
func (c *Client) FileModificationTime (fileName string) (t *time.Time, err error) {
	var dir, file string
	dir = c.toAbsolutePath(fileName) /* Normalize to an absolute path directory */
	_, file = c.extractPathElements(fileName) /* Extract file name */
	command := c.Request(NewCommand("mdtm", path.Join(dir, file), Status.FileStatus))
	t, err = c.parseTimeVal(command.Response().Message())

	if err != nil {
		command.AddError(err)
	}

	return t, command.LastError()
}

/* Asks the server for the file size */
func (c *Client) FileSize (fileName string) (size int, err error) {
	var dir, file string
	dir = c.toAbsolutePath(fileName) /* Normalize to an absolute path directory */
	_, file = c.extractPathElements(fileName) /* Extract file name */
	command := c.Request(NewCommand("size", path.Join(dir, file), Status.FileStatus))
	size = Status.ToInt([]byte(command.Response().Message()))

	return size, command.LastError()
}

/* Gives the ability to define the file structure (F, R, P) */
func (c *Client) FileStructure(fileStructureType string) (bool, error) {
	fileStructureType = strings.TrimSpace(strings.ToUpper(fileStructureType))
	if fileStructureType != FILESTRUCT_File && fileStructureType != FILESTRUCT_Record && fileStructureType != FILESTRUCT_Page {
		return false, ERR_InvalidFileStructure
	}

	command := c.Request(NewCommand("stru", fileStructureType, Status.PositiveCompletion))

	if command.Success() {
		c.settings.Get(OPT_FileStructure).Set(fileStructureType)
	}

	return command.Success(), command.LastError()
}

/* Request help from the server */
func (c *Client) Help() (helpMessage string, err error) {
	command := c.Request(NewCommand("help", CONST_EmptyString, Status.HelpMessage))
	return command.Response().Message(), command.LastError()
}

/* Request help for a specific command */
func (c *Client) HelpWith(commandName string) (helpMessage string, err error) {
	command := c.Request(NewCommand("help", commandName, Status.HelpMessage))
	return command.Response().Message(), command.LastError()
}

/* Request server to expose the user to the content's of the specified virtual host */
func (c *Client) Host(virtualHostDesired string) (bool, error) {
	/* If a user is authenticated, try to reinitialize the connection */
	if c.settings.Get(OPT_Authenticated).Is(true) {
		if ok, _ := c.Reinitialize(); !ok {
			/* Could not reinitialize the connection, notify the user to create a new connection */
			return false, ERR_SelectVirtualHostBeforeAuth
		}
	}

	command := c.Request(NewCommand("host", virtualHostDesired, 0))
	return command.Success(), command.LastError()
}

/* Request the server to use the specified language for response messages */
func (c *Client) Language(language string) (ok bool, err error) {
	command := c.Request(NewCommand("lang", language, Status.PositiveCompletion))
	return command.Success(), command.LastError()
}

/* Query the server to determine the currently supported languages (FEAT reuse) */
func (c *Client) LanguagesSupported() (languages []string, err error) {
	var langsString string
	features, err := c.Features()

	if err == nil {
		langsString, _ = features["LANG"]

		for _, lang := range strings.Split(langsString, ";") {
			if strings.Contains(lang, "*") {
				lang = strings.Join(strings.Split(lang, "*"), CONST_EmptyString)
			}

			languages = append(languages, lang)
		}
	}

	return
}

/* List the contents of the current directory */
func (c *Client) ListWorkingDir() (list string, err error) {
	return c.List(c.toAbsolutePath(CONST_EmptyString))
}

/* List the contents of the specified file or directory */
func (c *Client) List(p string) (list string, err error) {
	var dir, file string

	if !c.inPassiveMode() {
		err = ERR_InvalidListCommand
		return
	}

	/* Normalize each file name */
	dir = c.toAbsolutePath(p)
	_, file = c.extractPathElements(p)
	p = path.Join(dir, file)

	/* Execute the list command if possible in the current context */
	command := c.Request(NewCommand("list", p, Status.DataConnectionClose))

	// TODO: Continue, grab data from the data connection once the current request finishes

	return CONST_EmptyString, command.LastError()
}

/* List contents of the current working directory by name */
func (c *Client) ListNamesWorkingDir() (list string, err error) {
	return c.ListNames(c.toAbsolutePath(CONST_EmptyString))
}

/* List the contents of the specified directory by name */
func (c *Client) ListNames(p string) (list string, err error) {
	var dir, file string

	if !c.inPassiveMode() {
		err = ERR_InvalidListCommand
		return
	}

	/* Normalize each file name */
	dir = c.toAbsolutePath(p)
	_, file = c.extractPathElements(p)
	p = path.Join(dir, file)

	/* Execute the list command if possible in the current context */
	command := c.Request(NewCommand("nlst", p, Status.DataConnectionClose))

	// TODO: Continue, grab data from the data connection once the current request finishes

	return CONST_EmptyString, command.LastError()
}

/* Aks the server to create a new directory with the specified name */
func (c *Client) MakeDirectory (p string) (ok bool, err error) {
	var dir, cwd string

	/* Normalize the directory name to an absolute path */
	dir = c.toAbsolutePath(p)
	cwd = c.settings.Get(OPT_CurrentDir).ToString()

	/* Check if the current working directory is part of the new desired path hierarchy */
	if idx := strings.LastIndex(dir, cwd); idx > - 1 {
		/* Restrain the the directory to it's relative format */
		dir = dir[idx + len(cwd):]
	}

	/* Recreate the entire path specified */
	parts := strings.Split(dir, "/")
	for _, k := range parts {
		command := c.Request(NewCommand("mkd", p, 0))
		ok = command.Success()

		if ok {
			/* Navigate to the newly created directory */
			ok, err = c.ChangeDirectory(k)

			if err != nil {
				err = ERR_InvalidMKDPath
				break
			}
		} else {
			err = ERR_InvalidMKDPath
			break
		}
	}

	/* Change the current working directory back to the original one */
	c.ChangeDirectory(cwd)

	return
}

/* Request the server a ready response to keep alive the connection (NOOP) */
func (c *Client) NoOP() (bool, error) {
	command := c.Request(NewCommand("noop", CONST_EmptyString, Status.PositiveCompletion))
	return command.Success(), command.LastError()
}

/* Gives the ability to set desired options for any of the FTP commands supporting options */
func (c *Client) Options(cmd string, options string) (bool, error) {
	cmd = Commands.ToStandardCommand(cmd)

	if !Commands.IsValid(cmd) {
		return false, fmt.Errorf(ERRF_InvalidCommandName, cmd)
	}

	command := c.Request(NewCommand("opts", cmd + " " + options, Status.PositiveCompletion))
	return command.Success(), command.LastError()
}

/* Reinitialize the connection to it's initial state, keeping open any data transfer connections */
func (c *Client) Reinitialize() (bool, error) {
	command := c.Request(NewCommand("rein", CONST_EmptyString, Status.Ready))
	return command.Success(), command.LastError()
}

/* Rename the specified file from it's original name to a newly selected name */
func (c *Client) Rename(fileName string, modifiedName string) (bool, error) {
	var dir, file string

	/* Normalize each file name */
	dir = c.toAbsolutePath(fileName)
	_, file = c.extractPathElements(fileName)
	fileName = path.Join(dir, file)

	dir = c.toAbsolutePath(modifiedName)
	_, file = c.extractPathElements(modifiedName)
	modifiedName = path.Join(dir, file)

	/* Execute the file renaming commands sequence */
	_, command := c.Sequence(
		NewCommand("rnfr", fileName, Status.FileActionPending),
		NewCommand("rnto", modifiedName, Status.FileActionOk),
	)

	return command.Success(), command.LastError()
}

/* Impose the specified representation type to the server */
func (c *Client) RepresentationType(representationType string, typeParameter interface {}) (bool, error) {
	var formatControl, byteSize bool
	var command *Command.Command

	/* Check if the specified type is supported */
	if _, ok := RepresentationTypes[representationType]; !ok {
		/* Unsupported representation type */
		return false, ERR_InvalidType
	}

	/* Determine parameter type */
	switch typeParameter.(type) {
	case string:
		/* Text representation type (A & E) format control */
		formatControl = true
	case int:
		/* Local byte Byte size type (L) */
		byteSize = true
	}

	if formatControl {
		/* Check if the specified type and format control represent a valid combination */
		if len(typeParameter.(string)) == 0 {
			/* Default to non print format control */
			typeParameter = FMTCTRL_NonPrint
		} else {
			aux := RepresentationTypes[representationType]
			if _, ok := aux[typeParameter.(string)]; !ok {
				return false, ERR_InvalidFMTCTRL
			}
		}

		/* Do not request the server if neither the representation, nor the format changed */
		if c.settings.Get(OPT_DataType).Is(representationType) && c.settings.Get(OPT_FormatControl).Is(typeParameter.(string)) {
			return true, nil
		}

		command = c.Request(NewCommand("type", representationType + " " + typeParameter.(string), Status.PositiveCompletion))

		if command.Success() {
			/* Remember the current data type and format control */
			c.settings.Get(OPT_DataType).Set(representationType)
			c.settings.Get(OPT_FormatControl).Set(typeParameter.(string))
			c.settings.Get(OPT_ByteSize).Reset()
		}
	} else  if byteSize {
		/* Local byte Byte size type */
		if typeParameter.(int) < 1 {
			return false, ERR_InvalidByteSize
		}

		command = c.Request(NewCommand("type", TYPE_LocalByte + " " + strconv.Itoa(typeParameter.(int)), Status.PositiveCompletion))

		if command.Success() {
			/* Remember the byte size */
			c.settings.Get(OPT_ByteSize).Set(strconv.Itoa(typeParameter.(int)))
			c.settings.Get(OPT_DataType).Reset()
			c.settings.Get(OPT_FormatControl).Reset()
		}
	} else {
		/* Image type (binary) */
		command = c.Request(NewCommand("type", TYPE_Image, Status.PositiveCompletion))

		if command.Success() {
			c.settings.Get(OPT_ByteSize).Reset()
			c.settings.Get(OPT_DataType).Reset()
			c.settings.Get(OPT_FormatControl).Reset()
		}
	}

	return command.Success(), command.LastError()
}

/* Gets the server current status */
func (c *Client) ServerStatus() (status string, err error) {
	command := c.Request(NewCommand("stat", CONST_EmptyString, Status.SystemStatus))
	return command.Response().Message(), command.LastError()
}

/* Gets the server's status for the current file transfer */
//func (c *Client) ServerTransferStatus() (status string, err error) {
//	// TODO
//}

/* Equivalent to list but on the control connection. //TODO: rename and implement parser */
func (c *Client) ServerStatusList(fileName string) (string, error) {
	var dir, file string
	dir = c.toAbsolutePath(fileName) /* Normalize to an absolute path directory */
	_, file = c.extractPathElements(fileName) /* Extract file name */
	command := c.Request(NewCommand("stat", path.Join(dir, file), Status.FileStatus))
	return command.Response().Message(), command.LastError()
}

/* Gives the ability to specify site parameters to the server */
func (c *Client) SiteParameters(params string) (bool, error) {
	command := c.Request(NewCommand("site", params, Status.PositiveCompletion))
	return command.Success(), command.LastError()
}

/* Adds specified account information to the client instance's settings */
func (c *Client) SetAccountData(accountInfo string) {
	c.settings.Get(OPT_Account).Set(accountInfo)
}

/* Allows mounting of a different file system data structure without altering login or accounting information  */
func (c *Client) StructureMount(p string) (bool, error) {
	var dir, file string
	dir = c.toAbsolutePath(p) /* Normalize to an absolute path directory */
	_, file = c.extractPathElements(p) /* Extract file name */
	command := c.Request(NewCommand("smnt", path.Join(dir, file), Status.FileActionOk))
	return command.Success(), command.LastError()
}

/* Server system type */
func (c *Client) SystemType() (string, error) {
	command := c.Request(NewCommand("syst", CONST_EmptyString, Status.NAMEType))
	return command.Response().Message(), command.LastError()
}

/* Puts the client in extended passive mode */
func (c *Client) ToExtendedPassiveMode() (ok bool, err error) {
	if c.settings.Get(OPT_ExtendedPassive).Is(true) {
		/* Client in extended passive mode, ignore the new request */
		return true, err
	}

	command := c.Request(NewCommand("epsv", CONST_EmptyString, Status.ExtendedPassiveMode))
	addr := Address.FromExtendedPortSpecifier(command.Response().Message())
	if nil == addr {
		command.AddError(ERR_InvalidDataConn)
	}

	if command.Success() {
		/* Remember the new data connection address */
		addr2 := Address.FromConnection(c.connection)
		addr2.Port = addr.Port
		c.dataAddr = addr2.ToTCPAddr()

		/* Establish a new data connection with the server using the specified port to verify availability */
		conn, err := net.DialTCP(CONST_ClientNetwork, nil, c.dataAddr)

		/* Mark the connection as being in passive mode */
		if err == nil {
			/* Reset passive mode flag, mark extended passive mode as active and close the test connection */
			c.settings.Get(OPT_PassiveMode).Reset()
			c.settings.Get(OPT_ExtendedPassive).Set(true)
			conn.Close()
		} else {
			/* Error connecting to remote server on data link */
			command.AddError(ERR_InvalidDataConn)
		}
	}

	return command.Success(), command.LastError()
}

/* Puts the client in long passive mode (this command is marked as obsolete in IANA commands extension list,
 reuse of extended passive mode) */
func (c *Client) ToLongPassiveMode() (ok bool, err error) {
	return c.ToExtendedPassiveMode()
}

/* Puts the client in passive mode */
func (c *Client) ToPassiveMode() (ok bool, err error) {
	if c.settings.Get(OPT_PassiveMode).Is(true) {
		/* Client in passive mode, ignore the new request */
		return true, err
	}

	command := c.Request(NewCommand("pasv", CONST_EmptyString, Status.PassiveMode))

	if command.Success() {
		/* Extract the server address and port */
		addr := Address.FromPortSpecifier(command.Response().Message())

		if nil == addr {
			/* Insuficcient data to determine the port part of a data connection port response */
			command.AddError(ERR_InvalidIpAndPortRepr)
		}

		if command.Success() {
			/* Register the new dataAddr */
			c.dataAddr = addr.ToTCPAddr()

//			/* Establish data connection to verify dataAddr validity */
//			conn, err := net.DialTCP(CONST_ClientNetwork, nil, c.dataAddr)
//
//			/* Mark the connection as being in passive mode */
//			if err == nil {
				/* Reset extended passive mode flag, mark passive mode as active and close the test connection */
				c.settings.Get(OPT_ExtendedPassive).Reset()
				c.settings.Get(OPT_PassiveMode).Set(true)
//				conn.Close()
//			} else {
//				/* Error connecting to remote server on data link */
//				command.AddError(ERR_InvalidDataConn)
//			}
//		} else {
//			command.AddError(err)
		}
	}

	return command.Success(), command.LastError()
}

/* Changes the current working directory to it's parent directory */
func (c *Client) ToParentDirectory() (bool, error) {
	command := c.Request(NewCommand("cdup", CONST_EmptyString, Status.FileActionOk))
	return command.Success(), command.LastError()
}

/* Change the server transfer mode */
func (c *Client) TransferMode(mode string) (bool, error) {
	mode = strings.TrimSpace(strings.ToUpper(mode))

	/* Check transfer mode type */
	if mode != TRANSFER_Stream && mode != TRANSFER_Compressed && mode != TRANSFER_Block {
		return false, ERR_InvalidTransferMode
	}

	command := c.Request(NewCommand("mode", mode, Status.PositiveCompletion))

	/* Remember the current transfer mode */
	if command.Success() {
		c.settings.Get(OPT_TransferMode).Set(mode)
	}

	return command.Success(), command.LastError()
}

/* Retrieve the current working directory on the server */
func (c *Client) WorkingDirectory() (dir string, err error) {
	var start, end int = -1, -1
	var sep rune = '"'

	command := c.Request(NewCommand("pwd", CONST_EmptyString, Status.Pathname))
	dir = command.Response().Message()

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
		dir = dir[start + 1:end]
	} else {
		dir = CONST_EmptyString
		command.AddError(ERR_NoPWDResult)
	}

	return dir, command.LastError()
}
