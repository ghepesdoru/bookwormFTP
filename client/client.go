package client

import(
	"fmt"
	Address 		"github.com/ghepesdoru/bookwormFTP/core/addr"
	ClientCommands 	"github.com/ghepesdoru/bookwormFTP/client/commands"
	Credentials 	"github.com/ghepesdoru/bookwormFTP/core/credentials"
	Features 		"github.com/ghepesdoru/bookwormFTP/core/parsers/features"
	FileManager		"github.com/ghepesdoru/bookwormFTP/core/fileManager"
	Logger			"github.com/ghepesdoru/bookwormFTP/core/logger"
	Path			"path"
	Resources 		"github.com/ghepesdoru/bookwormFTP/core/parsers/resource"
	Requester 		"github.com/ghepesdoru/bookwormFTP/client/requester"
	Settings 		"github.com/ghepesdoru/bookwormFTP/client/settings"
	Status 			"github.com/ghepesdoru/bookwormFTP/core/codes"
)

/* Constants definition */
const (
	EmptyString			= ""
	RootDir				= "/"

	/* Connection option names */
	OPT_DebugMode	 	= "debug"
	OPT_System			= "system"
	OPT_Disconnected 	= "disconnected"
	OPT_LoggedIn		= "logged_in"
	OPT_PassiveMode		= "passive"
	OPT_ExtendedPassive = "extended_passive"
	OPT_CurrentDir		= "cwd"
	OPT_Account			= "account"
	OPT_AccountEnabled 	= "account_active"
	OPT_TransferMode	= "transfer_mode"
	OPT_DataType		= "data_type"
	OPT_FormatControl	= "format_control"
	OPT_ByteSize		= "byte_size"
	OPT_FileStructure	= "file_structure"
)

var (
	ERR_NonRetrievable	= fmt.Errorf("Non retrievable resource.")
)

/* BookwormFTP Client type definition */
type Client struct {
	Commands	*ClientCommands.Commands
	requester	*Requester.Requester
	credentials *Credentials.Credentials
	currentDir	string
	settings 	*Settings.Settings
	features	*Features.Features
	Resources	*Resources.Resource
	localFM		*FileManager.FileManager
}

/* Instantiates a new client (IPv4 preferred), and takes all possible actions based on address url */
func NewClient(address string) (client *Client, err error) {
	var dir, system string

	client, err = newClient(address, Address.IPvAny)

	if err != nil {
		return
	}

	/* Authenticate with USER and PASS */
	_, err = client.LogIn(client.credentials)

	if err != nil {
		return
	}

	/* Get system type */
	if system, err = client.System(); err == nil {
		client.settings.Add(OPT_System, system)
	}

	/* Grab supported features list */
	if _, err = client.Features(); err != nil {
		return
	}

	/* Get the current directory */
	dir, err = client.Commands.PWD()
	if err == nil {
		client.currentDir = dir
	}

	/* Check for initial path, and navigate there if available */
	dir, _ = client.requester.GetInitialPath()

	/* Enforce transfer parameters defaults (as specified by RFC959) */
	client.RepresentationType(ClientCommands.TYPE_Ascii, ClientCommands.FMTCTRL_NonPrint)
	client.TransferMode(ClientCommands.TRANSFER_Stream)
	client.FileStructure(ClientCommands.FILESTRUCT_File)

	if dir != EmptyString {
		_, err = client.ChangeDir(dir)
	} else {
		_, err = client.List()
	}

	return
}

/* Instantiates a new client and tarts downloading the specified Resources */
func NewDownload(address string) (client *Client, err error) {
	client, err = NewClient(address)

	if err != nil {
		return
	}

	/* Download the specified file */
	_, file := client.requester.GetInitialPath()
	_, err = client.Download(file)
	return
}

/* Instantiates a new client forcing IP version 4 */
func NewIPv4(address string) (*Client, error) {
	return newClient(address, Address.IPv4)
}

/* Instantiates a new client forcing IP version 6 */
func NewIPv6(address string) (*Client, error) {
	return newClient(address, Address.IPv6)
}

/* Changes the current working directory on the host */
func (c *Client) ChangeDir(path string) (ok bool, err error) {
	var dir string
	dir = c.toAbsolutePath(path)

	if dir == c.currentDir {
		ok = true
	} else {
		ok, err = c.Commands.CWD(dir)
		if ok {
			/* Keep track of the current working directory */
			c.currentDir = dir

			/* List the current dir */
			c.List()
		}
	}

	return ok, err
}

/* Changes the current working directory to it's container on the host */
func (c *Client) ChangeToParentDir() (bool, error) {
	if c.currentDir == RootDir {
		return true, nil
	}

	ok, err := c.Commands.CDUP()
	if ok {
		/* Keep track of the current working directory */
		c.currentDir = Path.Dir(Path.Clean(c.currentDir))
	}

	return ok, err
}

/* Get the current working directory on host */
func (c *Client) CurrentDir() string {
	return c.currentDir
}

/* Download the specified file */
func (c *Client) Download(fileName string) (ok bool, err error) {
	if !c.InPassiveMode() {
		_, err = c.PassiveMode()
		defer c.RestoreConnections();
	}

	fileName = c.toAbsolutePath(fileName)
	dir, file := c.extractPathElements(fileName)

	if ok, err = c.ChangeDir(dir); ok {
		if c.Resources.ContainsByName(file) {
			r := c.Resources.GetContentByName(file)

			if r != nil {
				if r.IsFile() {
					/* Download the specified file */
					if r.CanBeRetrieved() {
						if r.IsBinary() {
							c.RepresentationType(ClientCommands.TYPE_Image, nil)
						} else {
							c.RepresentationType(ClientCommands.TYPE_Ascii, ClientCommands.FMTCTRL_NonPrint)
						}

						if !c.localFM.ContainsFile(file) {
							ok, err = c.localFM.CreateFile(file)
						}

						if err == nil {
							_, err = c.localFM.Select(file)
							_, err = c.Commands.RETR(file, c.localFM.GetSelectionWriter())
							c.localFM.SelectionClear()
						}
					} else {
						err = ERR_NonRetrievable
					}
				} else {
					/* Download the entire directory */
					// TODO:
				}
			}
		}
	}

	return
}

/* Extracts the server supported features map */
func (c *Client) Features() (feat *Features.Features, err error) {
	if c.features == nil {
		feat, err = c.Commands.FEAT()

		if err == nil {
			c.features = feat
		}
	} else {
		feat = c.features
	}

	return
}

/* Gives the ability to modify the currently used file structure (STRU) */
func (c *Client) FileStructure(fileStructure string) (ok bool, err error) {
	if c.settings.Get(OPT_FileStructure).Is(fileStructure) {
		return true, nil
	}

	ok, err = c.Commands.STRU(fileStructure)
	if err == nil {
		c.settings.Get(OPT_FileStructure).Set(fileStructure)
	}
	return ok, err
}

/* Checks if the client is in any of the supported passive modes */
func (c *Client) InPassiveMode() bool {
	return c.settings.Get(OPT_PassiveMode).Is(true) || c.settings.Get(OPT_ExtendedPassive).Is(true)
}

/* Checks if the current client established a connection using IP version 4 */
func (c *Client) IsIPv4() bool {
	return c.requester.GetHostAddr().IPFamily == Address.IPv4
}

/* Lists the contents of the current directory */
func (c *Client) List() (*Resources.Resource, error) {
	return c.ListDir(c.currentDir)
}

/* Lists the contents of the specified directory */
func (c *Client) ListDir(dir string) (*Resources.Resource, error) {
	return c.list(c.toAbsolutePath(dir), false)
}

/* Lists the specified file properties */
func (c *Client) ListFile(fileAndPath string) (*Resources.Resource, error) {
	return c.list(c.toAbsolutePath(fileAndPath), true)
}

/* Uses one of the supported features to list a container's resources or the named resource's facts */
func (c *Client) list(path string, isFile bool) (res *Resources.Resource, err error) {
	var executed bool
	if !c.InPassiveMode() {
		_, err = c.PassiveMode()
		defer c.RestoreConnections();
	}

	if err == nil {
		if !isFile {
			/* Container listing */
			if c.features.Supports("MLSD") {
				res, err = c.Commands.MLSD(path)
				executed = true
			}

			if executed && err != nil {
				/* MLSD not supported, remove the feature from expected support and fallback on LIST */
				c.features.RemoveFeature("MLSD")
				executed = false
			}

			if !executed && c.features.Supports("LIST") {
				res, err = c.Commands.LIST(path)
			}
		} else {
			/* Single resource listing */
			if c.features.Supports("MLST") {
				res, err = c.Commands.MLST(path)
				executed = true
			}

			/* MLST not supported, remove the feature from expected support and fallback on LIST */
			if executed && err != nil {
				c.features.RemoveFeature("MLST")
				executed = false
			}

			if c.features.Supports("LIST") {
				res, err = c.Commands.LIST(path)
			}
		}
	}

	if err == nil {
		c.Resources = res
	}

	return
}

/* Log's in with client registered credentials (USER, PASS sequence) */
func (c *Client) LogIn(credentials *Credentials.Credentials) (bool, error) {
	_, command := c.requester.Sequence(
		ClientCommands.NewCommand("user", credentials.Username(), Status.UserNameOk),
		ClientCommands.NewCommand("pass", credentials.Password(), Status.UserLoggedIn, Status.AccountForLogin),
	)

	return command.Success(), command.LastError()
}

/* Puts the client in passive mode, and makes the client ready for accessing the data connection */
func (c *Client) PassiveMode() (bool, error) {
	return c.passiveMode(!c.IsIPv4() || c.features.Supports("EPSV"))
}

/* Puts the client in passive mode, forces usage of EPSV command */
func (c *Client) PassiveModeEPSV() (bool, error) {
	return c.passiveMode(true)
}

/* Impose the specified representation type to the server */
func (c *Client) RepresentationType(representationType string, typeParameter interface {}) (bool, error) {
	/* Do not request the server if neither the representation, nor the format changed */
	if c.settings.Get(OPT_DataType).Is(representationType) {
		if representationType == ClientCommands.TYPE_Ascii {
			if c.settings.Get(OPT_FormatControl).Is(typeParameter) {
				return true, nil
			}
		} else if representationType == ClientCommands.TYPE_LocalByte {
			if c.settings.Get(OPT_ByteSize).Is(typeParameter) {
				return true, nil
			}
		} else {
			/* Still I type. */
			return true, nil
		}
	}

	ok, err := c.Commands.TYPE(representationType, typeParameter)

	if ok {
		if representationType == ClientCommands.TYPE_Ascii {
			c.settings.Get(OPT_DataType).Set(representationType)
			c.settings.Get(OPT_FormatControl).Set(typeParameter.(string))
			c.settings.Get(OPT_ByteSize).Reset()
		} else if representationType == ClientCommands.TYPE_LocalByte {
			c.settings.Get(OPT_ByteSize).Set(typeParameter)
			c.settings.Get(OPT_DataType).Reset()
			c.settings.Get(OPT_FormatControl).Reset()
		} else {
			c.settings.Get(OPT_ByteSize).Reset()
			c.settings.Get(OPT_DataType).Reset()
			c.settings.Get(OPT_FormatControl).Reset()
		}
	}

	return ok, err
}

/* Restores the client connection settings to the defaults */
func (c *Client) RestoreConnections() {
	c.settings.Get(OPT_ExtendedPassive).Reset()
	c.settings.Get(OPT_PassiveMode).Reset()
}

/* Gets the system type */
func (c *Client) System() (string, error) {
	return c.Commands.SYST()
}

/* Gives the ability to define the desired data transfer mode */
func (c *Client) TransferMode(mode string) (bool, error) {
	if c.settings.Get(OPT_TransferMode).Is(mode) {
		return true, nil
	}

	ok, err := c.Commands.MODE(mode)
	if ok {
		c.settings.Get(OPT_TransferMode).Set(mode)
	}

	return ok, err
}

/* Instantiate a new client */
func newClient(address string, ipFamily int) (client *Client, err error) {
	var commands *ClientCommands.Commands
	var credentials *Credentials.Credentials
	var requester *Requester.Requester

	/* Create a new client instance based on specified IP version */
	if ipFamily != Address.IPvAny {
		if ipFamily == Address.IPv4 {
			requester, err = Requester.NewRequesterIPv4(address)
		} else {
			requester, err = Requester.NewRequesterIPv6(address)
		}

		if err != nil {
			return
		}

		commands = ClientCommands.NewCommands()
		_, err = commands.AttachRequester(requester)
	} else {
		commands, err = ClientCommands.NewCommandsProvider(address)
		if err != nil {
			return
		}

		requester = commands.Requester()
		if requester == nil {
			return
		}
	}

	if nil == err {
		credentials = requester.GetCredentials()
		client = &Client{commands, requester, credentials, RootDir, Settings.NewSettings(
			Settings.NewOption(OPT_DebugMode, true),
			Settings.NewOption(OPT_LoggedIn, false),
			Settings.NewOption(OPT_PassiveMode, false),
			Settings.NewOption(OPT_ExtendedPassive, false),
			Settings.NewOption(OPT_Account, EmptyString),
			Settings.NewOption(OPT_AccountEnabled, false),
			Settings.NewOption(OPT_TransferMode, ClientCommands.TRANSFER_Unspecified),
			Settings.NewOption(OPT_DataType, ClientCommands.TYPE_Unspecified),
			Settings.NewOption(OPT_FormatControl, ClientCommands.FMTCTRL_Unspecified),
			Settings.NewOption(OPT_ByteSize, 8),
			Settings.NewOption(OPT_FileStructure, ClientCommands.FILESTRUCT_Unspecified),
		), nil, nil, nil}

		/* Initialize a local file manager based on the client current working dir (localy) */
		client.localFM, err = FileManager.NewFileManager()

		/* Enable debugging */
		if client.settings.Get(OPT_DebugMode).Is(true) {
			requester.Logger = Logger.NewSimpleLogger()
		}
	}

	return
}

/* Extracts the current path parts (directories and file) from the specified input */
func (c *Client) extractPathElements(p string) (dir string, file string) {
	p = Path.Clean(p)

	if Path.Ext(p) == EmptyString && p != RootDir {
		p = p + "/"
	}

	return Path.Split(p)
}

/* Given a relative path, will concatenate it with the current working directory and normalize it */
func (c *Client) toAbsolutePath (d string) string {
	var f string
	if !Path.IsAbs(d) {
		/* Not an absolute path, concatenate the relative path to the working directory, and normalize them together  */
		if len(d) == 0 {
			d = c.currentDir
		} else {
			d = c.currentDir + RootDir + d
		}
		d, f = c.extractPathElements(d)
		d = Path.Join(d, f)
	}

	return d
}

/* Activates the passive mode if possible, and marks the internal options */
func (c *Client) passiveMode(epsv bool) (ok bool, err error) {
	if epsv {
		ok, err = c.Commands.EPSV()

		if ok {
			c.settings.Get(OPT_ExtendedPassive).Set(true)
			c.settings.Get(OPT_PassiveMode).Reset()
		}
	} else {
		ok, err = c.Commands.PASV()

		if ok {
			c.settings.Get(OPT_PassiveMode).Set(true)
			c.settings.Get(OPT_ExtendedPassive).Reset()
		}
	}

	return
}
