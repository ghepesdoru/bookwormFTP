package client

import(
	"fmt"
	"strconv"
	"net/url"
	Net "net"
	Address "github.com/ghepesdoru/bookwormFTP/core/addr"
	Credentials "github.com/ghepesdoru/bookwormFTP/core/credentials"
	Reader "github.com/ghepesdoru/bookwormFTP/core/reader"
	Settings "github.com/ghepesdoru/bookwormFTP/client/settings"
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
	ERR_InvalidHostUrl = fmt.Errorf("Invalid host URL.")
	ERR_UnableToLookupHost = fmt.Errorf("Unable to lookup host address. The host might not support the specified IP version, or is currently unavailable.")
)

/* BookwormFTP Client type definition */
type Client struct {
	controlConnection	Net.Conn
	controlReader		*Reader.Reader
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

	/* Register the initial navigation path if required */
	// TODO: Take eventual files into account for cases when the user requires a quick download only.
	if len(navigateTo) > 0 {
		settings.Get(OPT_InitialPath).Set(navigateTo)
	}

	conn, err = Net.Dial(hostAddr.Network(), hostAddr.String())
	if err != nil {
		return /* Return with the original DIal generated error. */
	}

	/* Connected successfully */
	settings.Get(OPT_Connected).Set(true)

	/* Instantiate the new client */
	client = &Client{conn, Reader.NewReader(conn), hostAddr, nil, credentials, settings, "/"}

	// TODO: Continue from here. Grab actual cwd; authenticate, add commands, refactor getResponse to support reliable data connections (and digest control connection intermediary output)
	fmt.Println(client)

	return
}
