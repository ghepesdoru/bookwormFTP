package requester

import(
	"fmt"
	"strconv"
	"time"
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
	Status "github.com/ghepesdoru/bookwormFTP/core/codes"
)

const (
	/* Generic constants */
	DefaultRequesterProtocol = "tcp"
	DefaultHostPort = 21
	DefaultUserName = "anonymous"
	DefaultPassword	= ""
	DefaultDataPort = -1
	DefaultWorkingDir = "/"
	EmptyString		= ""
	CommandRetries 	= 3
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
	ERR_InvalidDataAddr			= fmt.Errorf("Invalid data connection Addr.")

	/* Error formats */
	ERRF_InvalidCommandName = "Command error: Unrecognized command %s."
	ERRF_InvalidCompletionStatus = "Command error: %s completed without meeting any of the %s status. Completion status: %d, completion message %s"
	ERRF_InvalidCommandOutOfSequence = "Command error: %s could not complete. Use a sequence for fequential commands. Intermediary status: %d, message: %s"
	ERRF_CommandMaxRetries = "Command error: %s reached the maximum number of retries. Transient Negative Completion reply status %d, message: %s"
	ERRF_CommandFailure = "Command failure: %d %s"
	ERRF_MissingPortInHost = "missing port in address"
)

/* BookwormFTP Requester type definition */
type Requester struct {
	controlConnection	Net.Conn
	controlReader		*Reader.Reader
	dataReader			*Reader.Reader
	hostAddress			*Address.Addr
	dataAddress			*Address.Addr
	credentials			*Credentials.Credentials
	initDir				string
	initFile			string
	connected			bool
	ready				bool
}

/* Generates a new Requester using any of the supported ip versions. (IPv4 first) */
func NewRequester(hostURL string) (r *Requester, err error) {
	r, err = NewRequesterIPv4(hostURL)
	if err != nil {
		r, err = NewRequesterIPv6(hostURL)
	}

	return
}

/* Instantiate a new Requester that can exclusively use the IPv4 */
func NewRequesterIPv4(hostURL string) (*Requester, error) {
	return preprocessRequesterBuild(hostURL, Address.IPv4)
}

/* Instantiate a new Requester that can exclusively use the IPv6 */
func NewRequesterIPv6(hostURL string) (*Requester, error) {
	return preprocessRequesterBuild(hostURL, Address.IPv6)
}

func (r *Requester) GetHostAddr() (*Address.Addr) {
	return Address.FromConnection(r.controlConnection)
}

/* Getter for the current requester registered hostURL path segments */
func (r *Requester) GetInitialPath() (string, string) {
	return r.initDir, r.initFile
}

/* Checks if the current requester is connected */
func (r *Requester) IsConnected() bool {
	return r.connected
}

/* Checks if the current requester is ready */
func (r *Requester) IsReady() bool {
	return r.ready
}

/* Register data address */
func (r *Requester) RegisterDataAddr(addr *Address.Addr) (bool, error) {
	if nil == addr {
		return false, ERR_InvalidDataAddr
	}

	/* Correct Addr that only specify a port */
	if addr.IP == nil {
		addr2 := r.GetHostAddr()
		addr.IP = addr2.IP
		addr.IPFamily = addr2.IPFamily
	}

	r.dataAddress = addr
	return true, nil
}

/* Make a request to the server */
func (r *Requester) Request(command *Command.Command) (*Command.Command) {
	r.execute(command, false, true, CommandRetries)
	return command
}

/* Make a data request to the server */
func (r *Requester) RequestData(command *Command.Command) (*Command.Command, []byte) {
	return r.executeDataCommand(command)
}

/* Make a sequence of requests */
func (r *Requester) Sequence(commands ...*Command.Command) (bool, *Command.Command) {
	return r.sequence(commands)
}

/* Prepares required data for the new Requester instance to be build */
func preprocessRequesterBuild(hostURL string, ipFamily int) (*Requester, error) {
	var host, path string
	var port int
	var credentials *Credentials.Credentials
	var err error
	var addr *Address.Addr

	if host, port, path, credentials, err = getHostParsedUrl(hostURL); err == nil {
		if addr, err = getHostAddr(host, port, ipFamily); err == nil {
			return buildRequester(addr, credentials, path)
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

/* Private Requester builder */
func buildRequester(hostAddr *Address.Addr, credentials *Credentials.Credentials, navigateTo string) (requester *Requester, err error) {
	var conn Net.Conn
	var dir, file string

	conn, err = Net.Dial(hostAddr.Network(), hostAddr.String())
	if err != nil {
		return /* Return with the original Dial generated error. */
	}

	/* Register the initial navigation path if available */
	if len(navigateTo) > 0 {
		navigateTo = "/" + navigateTo
		navigateTo = Path.Clean(navigateTo)
		dir, file = Path.Split(navigateTo)

		if len(dir) == 0 || dir == DefaultWorkingDir {
			dir = EmptyString
		}
	}

	/* Instantiate the new Requester */
	requester = &Requester{conn, Reader.NewReader(conn), nil, hostAddr, nil, credentials, dir, file, true, false}

	/* Grab server greeting, and check for server ready status */
	welcomeMessage, _ := requester.getResponse()
	if welcomeMessage != nil {
		if Status.Ready == welcomeMessage.Status() {
			/* Server ready */
			requester.ready = true
		}
	}

	return
}

/* Reads a response taking the data connection into account. */
func (r *Requester) getResponse() (response *Response.Response, err error) {
	var parser *Parser.Parser = Parser.NewParser()
	var raw []byte = r.controlReader.Get()

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
func (r *Requester) establishDataConnection() (conn Net.Conn, err error) {
	if r.dataAddress != nil {
		conn, err = Net.Dial(r.dataAddress.Network(), r.dataAddress.String())
		return
	} else {
		err = ERR_InvalidDataAddress
	}

	return nil, ERR_CanNotEstablishDataConn
}

/* Start listening for incoming data in the data channel */
func (r *Requester) listenDataChannel() (ok bool, err error) {
	var dataConnection Net.Conn

	if r.dataReader != nil {
		/* Stop the old reader from reading, and instantiate a new one for the current data connection. */
		r.dataReader.StopReading()
	}

	dataConnection, err = r.establishDataConnection()

	if err == nil {
		r.dataReader = Reader.NewReader(dataConnection)
		ok = true
	}

	return ok, err
}

/* Close the data channel and return all collected data */
func (r *Requester) closeDataChannel() (ok bool, data []byte) {
	if r.dataReader != nil {
		data = r.dataReader.Get()
		r.dataReader.StopReading()
		r.dataReader = nil
		ok = true
	}

	return ok, data
}

/* Makes requests to the server based on provided Command contents */
func (r *Requester) request(command *Command.Command) (bool, error) {
	var EOL []byte = []byte("\r\n")

	/* Send the request to the server */
	n, err := r.controlConnection.Write(append(command.Byte(), EOL...))
	return n > 0, err
}

/* Executes the specified command listening on the data connection. */
func (r *Requester) executeDataCommand(command *Command.Command) (*Command.Command, []byte) {
	var err error

	/* Listen for incoming data on the data connection (if required) */
	_, err = r.listenDataChannel()

	if err != nil {
		command.AddError(err)
		return command, []byte{}
	}

	r.execute(command, false, true, CommandRetries)
	_, data := r.closeDataChannel()

	fmt.Println("Command status:", command.Success(), command.Response(), command.LastError())

	return command, data
}

/* Executes a command (wrapper around request, takes care of response reading, error handling, and is status aware) */
func (r *Requester) execute(command *Command.Command, isSequence bool, execute bool, leftRetries int) {
	var err error

	if command.Name() == Commands.UnknownCommand {
		command.AddError(Commands.ERR_InvalidCommandName)
	}

	if !r.IsReady() {
		/* Do not make requests on closed connections */
		command.AddError(ERR_ServerNotReady)
		return
	}

	/* Execute the command */
	if execute {
		_, err = r.request(command)
		execute = false

		/* Error communicating to server */
		if err != nil {
			command.AddError(err)
			return
		} else {
			/* Get the server response */
			command.AttachResponse(r.getResponse())
		}
	} else {
		t := time.Now()
		/* Block until the server responds */
		for len(r.controlReader.Peek()) == 0 {
			time.Sleep(100 * time.Millisecond)
			fmt.Println("Waiting for server response since: ", time.Since(t))
		}

		/* Get the server response */
		command.AttachResponse(r.getResponse())
	}

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
		r.execute(command, isSequence, execute, leftRetries)
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
				r.execute(command, isSequence, true, leftRetries - 1)
				return
			}
		}
	} else if first == 5 {
		/* Permanent Negative Completion reply - failure. Forward the server error message */
		command.AddError(fmt.Errorf(ERRF_CommandFailure, status, command.Response().Message()))
	}
}

/* Executes a specified sequence of commands */
func (r *Requester) sequence(commands []*Command.Command) (ok bool, last *Command.Command) {
	var leftRetries int = CommandRetries
	var retry bool = false

	for leftRetries > 0 {
		for _, command := range commands {
			last = r.Request(command)

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