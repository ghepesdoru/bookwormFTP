package client

import (
	"fmt"
	"net"
	"regexp"
	"net/url"
	"strings"
	Commands "github.com/ghepesdoru/bookwormFTP/core/commands"
	Status "github.com/ghepesdoru/bookwormFTP/core/codes"
	Reader "github.com/ghepesdoru/bookwormFTP/core/readers/reader"
	Response "github.com/ghepesdoru/bookwormFTP/core/response"
	Parser "github.com/ghepesdoru/bookwormFTP/core/parsers/responseParser"
	Settings "github.com/ghepesdoru/bookwormFTP/client/settings"
	Credentials "github.com/ghepesdoru/bookwormFTP/core/credentials"

)

/* Constants definition */
const (
	CONST_ClientNetwork = "tcp"
	CONST_ServerPort 	= 21
	CONST_DefaultUser 	= "anonymous"
	CONST_DefaultPass	= ""
	CONST_DefaultResponse = ""
	CONST_EmptyString	= ""
	CONST_CommandRetries= 3
	CONST_DataPort	 	= -1

	/* Connection option names */
	OPT_Connected 		= "connected"
	OPT_ServerReady 	= "ready"
	OPT_InitialPath 	= "initial_path"
	OPT_Disconnected 	= "disconnected"
	OPT_Authenticated 	= "logged_in"
	OPT_DataPort		= "client_data_port"
	OPT_PassiveMode		= "passive"
)

/* Default errors definition */
var (
	ERR_InvalidClientAddress = fmt.Errorf("Invalid client address.")
	ERR_ResponseParsingError = fmt.Errorf("An error triggered while parsing the server response.")
	ERR_UnconsumedResponses	 = fmt.Errorf("Acumulation of unconsummed responses from the server.")
	ERR_NoServerResponse	 = fmt.Errorf("Response error: Unable to fetch a response from server at this time.")
	ERR_RestartSequence		 = fmt.Errorf("Restart sequence.")
	ERR_ServerNotReady		 = fmt.Errorf("Server is not connected/disconnected or otherwise unavailable.")
	ERR_ReinNotImplemented	 = fmt.Errorf("Server state reinitialization not supported. (REIN)")
	ERR_NoServerFeatures	 = fmt.Errorf("Server supported features unavailable.")
	ERR_NoPWDResult			 = fmt.Errorf("Could not determine the current working directory path.")
	ERR_InvalidListCommand	 = fmt.Errorf("Unable to list requested content. Please consider putting the client in passive mode or providing a client port.")

	/* Error formats */
	ERRF_InvalidCommandName = "Command error: Unrecognized command %s."
	ERRF_InvalidCompletionStatus = "Command error: %s completed without meeting the %d status. Completion status: %d, completion messege %s"
	ERRF_InvalidCommandOutOfSequence = "Command error: %s could not complete. Use a sequence for fequential commands. Intermediary status: %d, message: %s"
	ERRF_CommandMaxRetries = "Command error: %s reached the maximum number of retries. Transient Negative Completion reply status %d, message: %s"
	ERRF_CommandFailure = "Command failure: %d %s"
	ERRF_MissingPortInHost = "missing port in address"
)

/* Other global declarations */
var (
	MatchHostAndPort = regexp.MustCompilePOSIX(`([0-9]{1,3}+,){5}+[0-9]{1,3}`)
	/* Matches a passive/port command address: ipv4,port (4 x 8bit + 2 x 8bit) */
)

/* BookwormFTP Client type definition */
type Client struct {
	connection net.Conn
	dataConn net.Conn
	reader *Reader.Reader
	credentials *Credentials.Credentials
	settings *Settings.Settings
}

/* Client Command type definition */
type Command struct {
	command string
	parameters string
	expectedStatus int
}

/* Client Command builder */
func NewCommand(command string, parameters string, expectedStatus int) *Command {
	command = Commands.ToStandardCommand(command)
	return &Command{command, parameters, expectedStatus}
}

func NewClient(address string) (client *Client, err error) {
	var urlData *url.URL
	var host string
	var credentials *Credentials.Credentials
	var conn net.Conn
	var settings *Settings.Settings = Settings.NewSettings(
		/* Define current connection settings with default values */
		Settings.NewOption(OPT_Connected, false),		/* There is a connection to the host */
		Settings.NewOption(OPT_ServerReady, false),		/* The server send it's welcome message? */
		Settings.NewOption(OPT_Disconnected, false),	/* A QUIT command was called? */
		Settings.NewOption(OPT_InitialPath, "/"),		/* Url specified initial path - will be deleted
														   once the client navigates to the specified path at
														   client creation time */
		Settings.NewOption(OPT_Authenticated, false),	/* A user is currently authenticated */
		Settings.NewOption(OPT_PassiveMode, false), 	/* Client is not in passive mode at connection time */
		Settings.NewOption(OPT_DataPort, CONST_DataPort), /* Register a default invalid data port */
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

	// TODO: Change working directory to the specified path

	return
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

/* Checks the server availability for command execution */
func (c *Client) serverReady() (ok bool) {
	return (
		c.settings.Get(OPT_Connected).Is(true) &&
		c.settings.Get(OPT_ServerReady).Is(true) &&
		c.settings.Get(OPT_Disconnected).Is(false))
}

/* Makes requests to the server based on provided Command contents */
func (c *Client) request(command *Command) (bool, error) {
	var toSend []byte
	var EOL []byte = []byte("\r\n")

	if len(command.parameters) > 0 {
		toSend = []byte(command.command + " " + command.parameters)
	} else {
		toSend = []byte(command.command)
	}

	/* Send the request to the server */
	n, err := c.connection.Write(append(toSend, EOL...))

	return n > 0, err
}

/* Executes a command (wrapper around request, takes care of response reading, error handling, and is status aware) */
func (c *Client) execute(command *Command, isSequence bool, execute bool, leftRetries int) (ok bool, err error, rMessage string) {
	/* Initialize variables to defaults */
	ok = false
	rMessage = CONST_DefaultResponse

	if command.command == Commands.CONST_UnknownCommand {
		/* Invalid command name */
		err = fmt.Errorf(ERRF_InvalidCommandName, command.command)
		return
	}

	if !c.serverReady() {
		/* Do not make requests on closed connections */
		err = ERR_ServerNotReady
		return
	}

	/* Execute the command */
	if execute {
		ok, err = c.request(command)

		/* Error communicating to server */
		if err != nil {
			return
		}
	}

	/* Get the server response */
	response, err := c.getResponse()

	if err != nil {
		/* An error took place while fetching or parsing the response */
		return
	} else if response == nil {
		/* Empty server response */
		err = ERR_NoServerResponse
		return
	}

	/* Remember the response message */
	rMessage = response.Message()

	/* Check response status to determine the execution completion */
	status := response.Status()

	/* Check relay status for next action */
	first := status / 100
//	second := (status / 10) % 10
	// TODO: Take the second digit into account in reply management also?

	if first == 1 {
		/* Positive Preliminary reply - wait for a new response */
		return c.execute(command, isSequence, false, leftRetries)
	} else if first == 2 {
		/* Positive Completion reply - action completed successfully, no matter of the expected status */
		if status != command.expectedStatus {
			err = fmt.Errorf(ERRF_InvalidCompletionStatus, command.command, command.expectedStatus, status, rMessage)
		}
	} else if first == 3 {
		/* Positive Intermediate reply - sequence of commands mandatory */
		if !isSequence {
			/* Error: Invalid single command. Use a sequence */
			err = fmt.Errorf(ERRF_InvalidCommandOutOfSequence, command.command, status, rMessage)
		}
	} else if first == 4 {
		if leftRetries == 0 {
			/* Stop the retry process. The acction failed to many times. */
			err = fmt.Errorf(ERRF_CommandMaxRetries, command.command, status, rMessage)
		} else {
			/* Transient Negative Completion reply - repeat the command(s) */
			if (isSequence) {
				/* Reset the sequence, this is a temporary error */
				err = ERR_RestartSequence
			} else {
				/* Try again to execute this command */
				return c.execute(command, isSequence, true, leftRetries - 1)
			}
		}
	} else if first == 5 {
		/* Permanent Negative Completion reply - failure. Forward the server error message */
		err = fmt.Errorf(ERRF_CommandFailure, status, rMessage)
	}

	if err != nil {
		ok = false
	}

	if ok {
		/* Command completed as expected - Debug point */
		fmt.Println("Successfull command: ", command.command, response)
	} else {
		/* Command failed - Debug point */
		fmt.Println("Error executing: ", command.command, err)
	}

	return
}

/* Executes a specified sequence of commands */
func (c *Client) sequence(commands []*Command) (ok bool, err error) {
	var leftRetries int = CONST_CommandRetries
	var retry bool = false

	for leftRetries > 0 {
		for _, command := range commands {
			ok, err, _ = c.execute(command, true, true, leftRetries)

			/* Take into consideration sequence retries */
			if err == ERR_RestartSequence {
				leftRetries -= 1
				retry = true
				break
			} else if ok != true || err != nil {
				break
			}
		}

		if retry != true {
			break
		}
	}

	return
}

/* Make a request to the server */
func (c *Client) Request(command *Command) (ok bool, err error, rMessage string) {
	return c.execute(command, false, true, CONST_CommandRetries)
}

/* Make a sequence of requests */
func (c *Client) Sequence(commands ...*Command) (ok bool, err error) {
	return c.sequence(commands)
}

/* Authenticate the user with provided credentials */
func (c *Client) Authenticate(credentials *Credentials.Credentials) (ok bool, err error) {
	var modified bool = false

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
		ok, err = c.Sequence(
			NewCommand("user", credentials.Username(), Status.UserNameOk),
			NewCommand("pass", credentials.Password(), Status.UserLoggedIn),
		)
	} else if modified {
		/* Reset the connection if supported (REIN) */
		ok, err = c.Reinitialize()

		if ok {
			/* Call Authenticate again after resetting the flag */
			c.settings.Get(OPT_Authenticated).Reset()
			return c.Authenticate(credentials)
		} else {
			/* Unable to reinitialize connection. */
			ok = false
			err = ERR_ReinNotImplemented
		}
	}

	if ok {
		/* Notify user authenticated */
		c.settings.Get(OPT_Authenticated).Set(true)
	}

	return
}

/* Changes the current directory on the server to the specified one */
func (c *Client) ChangeDirectory(path string) (ok bool, err error) {
	ok, err, _ = c.Request(NewCommand("cwd", path, Status.FileActionOk))
	return
}

/* Delete the specified file on the remote server */
func (c *Client) DeleteFile(fileName string) (ok bool, err error) {
	ok, err, _ = c.Request(NewCommand("dele", fileName, Status.FileActionOk))
	return
}

/* Disconnect Command functionality */
func (c *Client) Disconnect() (quitMessage string, err error) {
	var ok bool
	ok, err, quitMessage = c.Request(NewCommand("quit", CONST_EmptyString, Status.ClosingControlConnection))

	if ok {
		/* Notify the server disconnection */
		c.settings.Get(OPT_Disconnected).Set(true)
		c.settings.Get(OPT_ServerReady).Reset()
		c.settings.Get(OPT_Authenticated).Reset()

		/* Close the TCP connection */
		c.connection.Close()
		c.settings.Get(OPT_Connected).Reset()
	}

	return
}

/* Server supported features getter */
func (c *Client) Features() (features map[string]string, err error) {
	var raw string
	var parts []string
	var length int
	var ok bool

	ok, err, raw = c.Request(NewCommand("feat", CONST_EmptyString, Status.SystemStatus))

	if ok {
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
					features[aux[0]] = strings.Join(aux[1:], " ")
				} else {
					features[aux[0]] = CONST_EmptyString
				}
			}
		}
	}

	if len(features) == 0 {
		err = ERR_NoServerFeatures
		ok = false
	}

	return
}

/* Request help from the server */
func (c *Client) Help() (helpMessage string, err error) {
	_, err, helpMessage = c.Request(NewCommand("help", CONST_EmptyString, Status.HelpMessage))
	return
}

/* Request help for a specific command */
func (c *Client) HelpWith(command string) (helpMessage string, err error) {
	_, err, helpMessage = c.Request(NewCommand("help", command, Status.HelpMessage))
	return
}

func (c *Client) List() (list string, err error) {
	var port, pasv bool = true, true
	if c.settings.Get(OPT_DataPort).Is(CONST_DataPort) {
		/* No client port was specified */
		port = false
	}

	if c.settings.Get(OPT_PassiveMode).Is(false) {
		/* The client connection is not in passive mode */
		pasv = false
	}

	if port || pasv {
		/* Execute the list command if possible in the current context */
		_, err, list = c.Request(NewCommand("list", "/ripe", 0))
		fmt.Println("Enters here!!!")
	} else {
		/* Invalid command in current context */
		err = ERR_InvalidListCommand
	}

	return
}

/* Puts the client in passive mode */
func (c *Client) ToPassiveMode() (ok bool, err error) {
	if (c.settings.Get(OPT_PassiveMode).Is(true)) {
		/* Client in passive mode, ignore the new request */
		return true, err
	}

	var r string
	ok, err, r = c.Request(NewCommand("pasv", CONST_EmptyString, Status.PassiveMode))

	if ok {
		/* Extract the server address and port */
		aux := MatchHostAndPort.FindString(r)
		p := strings.Split(aux, ",")

		if len(p) == 6 {
			fmt.Println("connecting to: ",  strings.Join(p[0:4], ".") + ":" + strings.Join(p[4:], ""))
			c.dataConn, err = net.Dial("tcp", strings.Join(p[0:4], ".") + ":" + strings.Join(p[4:], ""))

			if err == nil {
				/* Mark the connection as being in passive mode */
				c.settings.Get(OPT_PassiveMode).Set(true)
			}
		} else {
			/* Error case. TODO: Continue */
			err = fmt.Errorf("Invalid server address and port for establishing data connection.")
		}
	}

	if err != nil {
		ok = false
	}

	return
}

/* Reinitialize the connection to it's initial state, keeping open any data transfer connections */
func (c *Client) Reinitialize() (ok bool, err error) {
	ok, err, _ = c.Request(NewCommand("rein", CONST_EmptyString, Status.Ready))
	return
}

/* Server system type */
func (c *Client) SystemType() (systemType string, err error) {
	_, err, systemType = c.Request(NewCommand("syst", CONST_EmptyString, Status.NAMEType))
	return
}

/* Changes the current working directory to it's parent directory */
func (c *Client) ToParentDirectory() (ok bool, err error) {
	ok, err, _ = c.Request(NewCommand("cdup", CONST_EmptyString, Status.FileActionOk))
	return
}

/* Retrieve the current working directory on the server */
func (c *Client) WorkingDirectory() (dir string, err error) {
	var start, end int = -1, -1
	var sep rune = '"'

	_, err, dir = c.Request(NewCommand("pwd", CONST_EmptyString, Status.Pathname))

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
		err = ERR_NoPWDResult
	}

	return
}
