package client

import(
	"fmt"
	"strconv"
	"net/url"
	Net "net"
	Path "path"
	Address "github.com/ghepesdoru/bookwormFTP/core/addr"
	Command "github.com/ghepesdoru/bookwormFTP/client/command"
	Commands "github.com/ghepesdoru/bookwormFTP/core/commands"
	Credentials "github.com/ghepesdoru/bookwormFTP/core/credentials"
	Parser "github.com/ghepesdoru/bookwormFTP/core/parser"
	Reader "github.com/ghepesdoru/bookwormFTP/core/reader"
	Response "github.com/ghepesdoru/bookwormFTP/core/response"
	Settings "github.com/ghepesdoru/bookwormFTP/client/settings"
	Status "github.com/ghepesdoru/bookwormFTP/core/codes"
)

const (
	/* Generic constants */
	DefaultClientProtocol = "tcp"
	DefaultHostPort = 21
	DefaultUserName = "anonymous"
	DefaultPassword	= ""
	DefaultDataPort = -1
	EmptyString		= ""
	CommandRetries 	= 3

	/* Connection option names */
	OPT_Connected 		= "connected"
	OPT_ServerReady 	= "ready"
	OPT_InitialPath 	= "initial_path"
	OPT_InitialFile		= "initail_file"
	OPT_Disconnected 	= "disconnected"
	OPT_Authenticated 	= "logged_in"
	OPT_PassiveMode		= "passive"
	OPT_ExtendedPassive = "extended_passive"
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

/* Error definitions */
var (
	ERR_InvalidHostUrl 			= fmt.Errorf("Invalid host URL.")
	ERR_UnableToLookupHost 		= fmt.Errorf("Unable to lookup host address. The host might not support the specified IP version, or is currently unavailable.")
	ERR_CanNotEstablishDataConn = fmt.Errorf("Unable to establish a data connection in the current context. Please consider establishing a passive mode or extended passive mode request.")
	ERR_InvalidDataAddress 		= fmt.Errorf("Invalid data address.")
	ERR_ResponseParsingError 	= fmt.Errorf("An error triggered while parsing the server response.")
	ERR_UnconsumedResponses	 	= fmt.Errorf("Acumulation of unconsummed responses from the server.")
	ERR_ServerNotReady		 	= fmt.Errorf("Server is disconnected or otherwise unavailable.")
	ERR_NoServerResponse	 	= fmt.Errorf("Unable to fetch a response from server at this time.")
	ERR_RestartSequence		 	= fmt.Errorf("Restart sequence.")

	/* Error formats */
	ERRF_InvalidCommandName = "Command error: Unrecognized command %s."
	ERRF_InvalidCompletionStatus = "Command error: %s completed without meeting any of the %s status. Completion status: %d, completion message %s"
	ERRF_InvalidCommandOutOfSequence = "Command error: %s could not complete. Use a sequence for fequential commands. Intermediary status: %d, message: %s"
	ERRF_CommandMaxRetries = "Command error: %s reached the maximum number of retries. Transient Negative Completion reply status %d, message: %s"
	ERRF_CommandFailure = "Command failure: %d %s"
	ERRF_MissingPortInHost = "missing port in address"
)

/* BookwormFTP Client type definition */
type Client struct {
	controlConnection	Net.Conn
	controlReader		*Reader.Reader
	dataReader			*Reader.Reader
	hostAddress			*Address.Addr
	dataAddress			*Address.Addr
	credentials			*Credentials.Credentials
	settings			*Settings.Settings
	workingDir			string
}

/* Instantiate a new client that can exclusively use the IPv4 */
func NewClientIPv4(hostURL string) (*Client, error) {
	return preprocessClientBuild(hostURL, Address.IPv4)
}

/* Instantiate a new client that can exclusively use the IPv6 */
func NewClientIPv6(hostURL string) (*Client, error) {
	return preprocessClientBuild(hostURL, Address.IPv6)
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

/* Prepares required data for the new client instance to be build */
func preprocessClientBuild(hostURL string, ipFamily int) (*Client, error) {
	var host, path string
	var port int
	var credentials *Credentials.Credentials
	var err error
	var addr *Address.Addr

	if host, port, path, credentials, err = getHostParsedUrl(hostURL); err == nil {
		if addr, err = getHostAddr(host, port, ipFamily); err == nil {
			return buildClient(addr, credentials, path)
		}
	}

	return nil, err
}

/* Normalizes URL parsing */
func getHostParsedUrl(hostURL string) (host string, port int, path string, credentials *Credentials.Credentials, err error) {
	var URLData *url.URL

	/* Extract url parts */
	URLData, err = url.Parse(hostURL)

	if err != nil {
		err = ERR_InvalidHostUrl
		return
	}

	/* Extract port */
	host, s, err := Net.SplitHostPort(URLData.Host)
	if err != nil {
		/* Fallback on default port - specified host does not specify a custom port */
		port = DefaultHostPort
		host = URLData.Host
		err = nil /* Clear the error. */
	} else {
		port, err = strconv.Atoi(s)

		if err != nil {
			/* Invalid port specified. Fallback on default */
			port = DefaultHostPort
		}
	}

	/* Check if any credentials are passed in the url */
	if URLData.User != nil {
		if password, ok := URLData.User.Password(); ok {
			credentials, err = Credentials.NewCredentials(URLData.User.Username(), password)
		} else {
			credentials, err = Credentials.NewCredentials(URLData.User.Username(), DefaultPassword)
		}
	}

	/* Use anonymous login as default for cases where credentials are not provided or otherwise invalid */
	if credentials == nil || err == Credentials.ERR_UsernameToShort {
		/* Create anonymous credentials */
		credentials, _ = Credentials.NewCredentials(DefaultUserName, DefaultPassword)
	}

	path = URLData.Path

	return
}

/* Extracts the host Address.Addr from the received hostURL, taking IP version selection into account. */
func getHostAddr(host string, port int, ipFamily int) (*Address.Addr, error)  {
	var ip Net.IP = nil

	ips, err := Net.LookupIP(host)

	if err != nil {
		return nil, err
	}

	for _, i := range ips {
		if Address.IPv4 == ipFamily && Address.IsIPv4(&i) {
			ip = i
		} else if Address.IPv6 == ipFamily && !Address.IsIPv4(&i) {
			ip = i
		}
	}

	if ip != nil {
		addr := Address.Addr{&ip, port, ipFamily}
		return &addr, nil
	}

	return nil, ERR_UnableToLookupHost
}

/* Private Client builder */
func buildClient(hostAddr *Address.Addr, credentials *Credentials.Credentials, navigateTo string) (client *Client, err error) {
	var conn Net.Conn
	var settings *Settings.Settings = Settings.NewSettings(
		/* Define current connection settings with default values */
		Settings.NewOption(OPT_Connected, false),					/* There is a connection to the host */
		Settings.NewOption(OPT_ServerReady, false),					/* The server send it's welcome message? */
		Settings.NewOption(OPT_Disconnected, false),				/* A QUIT command was called? */
		Settings.NewOption(OPT_Authenticated, false),				/* A user is currently authenticated */
		Settings.NewOption(OPT_PassiveMode, false), 				/* Is the Client in passive mode at the time */
		Settings.NewOption(OPT_ExtendedPassive, false), 			/* Is extended passive mode ? */
		Settings.NewOption(OPT_Account, EmptyString),			/* Default account */
		Settings.NewOption(OPT_AccountEnabled, false),				/* No active account */
		Settings.NewOption(OPT_TransferMode, TRANSFER_Unspecified), /* The connection has no specified transfer mode at this point */
		Settings.NewOption(OPT_DataType, TYPE_Ascii), 				/* Presume ASCII as default data type */
		Settings.NewOption(OPT_FormatControl, FMTCTRL_NonPrint), 	/* Presume non print format control */
		Settings.NewOption(OPT_ByteSize, 8),						/* Assume a 8 bit byte size */
		Settings.NewOption(OPT_FileStructure, FILESTRUCT_File), 	/* Default to the file structure of file */
	)

	conn, err = Net.Dial(hostAddr.Network(), hostAddr.String())
	if err != nil {
		return /* Return with the original Dial generated error. */
	}

	/* Connected successfully */
	settings.Get(OPT_Connected).Set(true)

	/* Instantiate the new client */
	client = &Client{conn, Reader.NewReader(conn), nil, hostAddr, nil, credentials, settings, "/"}

	/* Register the initial navigation path if available */
	if len(navigateTo) > 0 {
		dir := client.toAbsolutePath(navigateTo) 	/* Normalize to an absolute path directory */
		_, file := client.extractPathElements(navigateTo) /* Extract file name */

		if len(dir) > 0 && dir != client.workingDir {
			client.settings.Get(OPT_InitialPath).Set(dir)
		}

		if len(file) > 0 {
			client.settings.Get(OPT_InitialFile).Set(file)
		}
	}

	/* Grab server greeting, and check for server ready status */
	welcomeMessage, _ := client.getResponse()
	if welcomeMessage != nil {
		if Status.Ready == welcomeMessage.Status() {
			/* Server ready */
			settings.Get(OPT_ServerReady).Set(true)
		}
	}

	// TODO: Authenticate once the command will be available and navigate to OPT_InitialPath and download OPT_InitialFile if available.
	fmt.Println(client)
	fmt.Println(welcomeMessage)

	return
}

/* Reads a response taking the data connection into account. */
func (c *Client) getResponse() (response *Response.Response, err error) {
	var parser *Parser.Parser = Parser.NewParser()
	var raw []byte = c.controlReader.Get()

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

/* Initializes a new data connection based on the current dataAddress value */
func (c *Client) establishDataConnection() (conn Net.Conn, err error) {
	if c.inPassiveMode() {
		if c.dataAddress != nil {
			conn, err = Net.Dial(c.dataAddress.Network(), c.dataAddress.String())
		} else {
			err = ERR_InvalidDataAddress
		}
	}

	return nil, ERR_CanNotEstablishDataConn
}

/* Start listening for incoming data in the data channel */
func (c *Client) listenDataChannel() (ok bool, err error) {
	var dataConnection Net.Conn

	if c.dataReader != nil {
		/* Stop the old reader from reading, and instantiate a new one for the current data connection. */
		c.dataReader.StopReading()

		dataConnection, err = c.establishDataConnection()

		if err == nil {
			c.dataReader = Reader.NewReader(dataConnection)
			ok = true
		}
	}

	return ok, nil
}

/* Close the data channel and return all collected data */
func (c *Client) closeDataChannel() (ok bool, data []byte) {
	if c.dataReader != nil {
		data = c.dataReader.GetBlock()
		c.dataReader.StopReading()
		c.dataReader = nil
		ok = true
	}

	return ok, data
}

/* Checks if the client is in any of the passive modes */
func (c *Client) inPassiveMode() bool {
	if c.settings.Get(OPT_PassiveMode).Is(true) || c.settings.Get(OPT_ExtendedPassive).Is(true) {
		return true
	}

	return false
}

/* Checks the server availability for command execution */
func (c *Client) serverReady() (ok bool) {
	return (
			c.settings.Get(OPT_Connected).Is(true) &&
			c.settings.Get(OPT_ServerReady).Is(true) &&
			c.settings.Get(OPT_Disconnected).Is(false))
}

/* Extracts the current path parts (directories and file) from the specified input */
func (c *Client) extractPathElements(p string) (dir string, file string) {
	p = Path.Clean(p)
	return Path.Split(p)
}

/* Given a relative path, will concatenate it with the current working directory and normalize it */
func (c *Client) toAbsolutePath (d string) string {
	if !Path.IsAbs(d) {
		/* Not an absolute path, concatenate the relative path to the working directory, and normalize them together  */
		d = c.workingDir + "/" + d
		d, _ = c.extractPathElements(d)
	}

	return d
}

/* Makes requests to the server based on provided Command contents */
func (c *Client) request(command *Command.Command) (bool, error) {
	var EOL []byte = []byte("\r\n")

	/* Send the request to the server */
	n, err := c.controlConnection.Write(append(command.Byte(), EOL...))

	return n > 0, err
}

/* Executes the specified command listening on the data connection. */
func (c *Client) executeDataCommand(command *Command.Command) (data []byte) {
	var err error

	/* Listen for incoming data on the data connection (if required) */
	_, err = c.listenDataChannel()

	if err != nil {
		command.AddError(err)
		return
	}

	c.execute(command, false, true, CommandRetries)
	_, data = c.closeDataChannel()

	return
}

/* Executes a command (wrapper around request, takes care of response reading, error handling, and is status aware) */
func (c *Client) execute(command *Command.Command, isSequence bool, execute bool, leftRetries int) {
	var err error

	if command.Name() == Commands.UnknownCommand {
		command.AddError(Commands.ERR_InvalidCommandName)
	}

	if !c.serverReady() {
		/* Do not make requests on closed connections */
		command.AddError(ERR_ServerNotReady)
		return
	}

	/* Execute the command */
	if execute {
		_, err = c.request(command)
		execute = false

		/* Error communicating to server */
		if err != nil {
			command.AddError(err)
			return
		}
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

	if first == 1 {
		/* Positive Preliminary reply - wait for a new response */
		c.execute(command, isSequence, execute, leftRetries)
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
}

/* Executes a specified sequence of commands */
func (c *Client) sequence(commands []*Command.Command) (ok bool, last *Command.Command) {
	var leftRetries int = CommandRetries
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
