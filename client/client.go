package client

import(
	"fmt"
	Address 		"github.com/ghepesdoru/bookwormFTP/core/addr"
	ClientCommands 	"github.com/ghepesdoru/bookwormFTP/client/commands"
	Credentials 	"github.com/ghepesdoru/bookwormFTP/core/credentials"
	Features 		"github.com/ghepesdoru/bookwormFTP/core/parsers/features"
	FileManager		"github.com/ghepesdoru/bookwormFTP/core/fileManager"
	Logger			"github.com/ghepesdoru/bookwormFTP/core/logger"
	PathManager 	"github.com/ghepesdoru/bookwormFTP/core/pathManager"
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
	OPT_DownloadOverlap	= "download_overlap"
)

var (
	ERR_ReinNotImplemented	 = fmt.Errorf("Server state reinitialization not supported. (REIN)")
	ERR_LoginAccountRequired = fmt.Errorf("Please specify an account and restart the authentication sequence.")
)

type DownloadOverlapAction string
const (
	DO_OverWrite		DownloadOverlapAction = "overwrite"
	DO_CreateNew		DownloadOverlapAction = "create_new"
	DO_IgnoreExisting	DownloadOverlapAction = "ignore_existing"
)

var (
	ERR_NonRetrievable	= fmt.Errorf("Non retrievable resource.")
)

/* BookwormFTP Client type definition */
type Client struct {
	Commands	*ClientCommands.Commands
	requester	*Requester.Requester
	credentials *Credentials.Credentials
	path		*PathManager.PathManager
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
		client.path.ChangeCurrentDir(dir)
	}

	/* Check for initial path, and navigate there if available */
	dir, _ = client.requester.GetInitialPath()

	/* Enforce transfer parameters defaults (as specified by RFC959) */
	client.RepresentationType(ClientCommands.TYPE_Ascii, ClientCommands.FMTCTRL_NonPrint)
	client.TransferMode(ClientCommands.TRANSFER_Stream)
	client.FileStructure(ClientCommands.FILESTRUCT_File)

	fmt.Println("Current dir to change to", dir)

	if dir != EmptyString {
		fmt.Println("Changes the directory, and it shold list the contents")
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

	/* Impose incremental creation of copies for existing files by default */
	client.SetDownloadRuleCreateCopy()

	/* Download the specified file or directory */
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

/* Register account information */
func (c *Client) Account(accountInfo string) (ok bool, err error) {
	return c.account(accountInfo, false, false)
}

/* Changes the current working directory on the host */
func (c *Client) ChangeDir(path string) (ok bool, err error) {
	var dir string
	dir = c.path.ToCurrentDir(path)

	if dir == c.path.GetCurrentDir() {
		ok = true
	} else {
		ok, err = c.Commands.CWD(dir)
		if ok {
			/* Keep track of the current working directory */
			c.path.ChangeCurrentDir(dir)

			/* List the current dir */
			c.List()
		}
	}

	return ok, err
}

/* Changes the current working directory to it's container on the host */
func (c *Client) ChangeToParentDir() (bool, error) {
	if c.path.InRootDir() {
		return true, nil
	}

	ok, err := c.Commands.CDUP()
	if ok {
		/* Keep track of the current working directory */
		c.path.ChangeCurrentDir("./../")
	}

	return ok, err
}

/* Get the current working directory on host */
func (c *Client) CurrentDir() string {
	return c.path.GetCurrentDir()
}

/* Download the specified file */
func (c *Client) Download(fileName string) (ok bool, err error) {
	if !c.InPassiveMode() {
		_, err = c.PassiveMode()
		defer c.RestoreConnections();
	}

	fileName = c.path.ToCurrentDir(fileName)
	dir := c.path.SplitDir(fileName)
	file := c.path.SplitFile(fileName)

	if ok, err = c.ChangeDir(dir); ok {
		/* Use the last subdirectory as container for the downloaded content */
		if len(file) == 0 {
			/* Download the entire current directory */
			return c.downloadDir(dir, false)
		} else {
			if c.Resources.ContainsByName(file) {
				r := c.Resources.GetContentByName(file)

				if r != nil {
					if r.IsFile() {
						/* Download the specified file */
						return c.downloadFile(file)
					} else {
						/* Download the entire directory */
						return c.downloadDir(file, true)
					}
				}
			}
		}
	}

	return
}

/* Download a directory at a time */
func (c *Client) downloadDir(currentDir string, changePath bool) (ok bool, err error) {
	if currentDir != RootDir {
		if !c.localFM.ContainsDir(currentDir) {
			dirs := c.path.SplitDirList(currentDir)

			for _, d := range dirs {
				if d == EmptyString {
					continue
				}

				/* Recreate the entire path to the current directory */
				if ok, err = c.localFM.MakeDir(d); !ok {
					err = fmt.Errorf("Download error: Unable to create local directory %s. Original error: %s", d, err)
					return
				} else {
					if ok, err = c.localFM.ChangeDir("./" + d); !ok {
						err = fmt.Errorf("Download error: Unable to change the current path to the newly created directory %s. Original Error: %s.", d, err)
					}
				}
			}
		}
	}

	/* Change the remote host directory */
	if changePath {
		fmt.Println("Change to specified directory.")
		if ok, err = c.ChangeDir(currentDir); !ok {
			return
		}
	}

	if err == nil {
		for _, f := range c.Resources.Content {
			if nil == f || !f.IsChild() {
				continue
			}

			if f.IsDir() {
				ok, err = c.downloadDir(f.Name, true)
			} else {
				/* File */
				ok, err = c.downloadFile(f.Name)
			}

			if !ok {
				err = fmt.Errorf("Download error: Unable to download remote resource %s. Original error: %s", f.Name, err)
				return
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
	return c.ListDir(c.path.GetCurrentDir())
}

/* Lists the contents of the specified directory */
func (c *Client) ListDir(dir string) (*Resources.Resource, error) {
	return c.list(c.path.ToCurrentDir(dir), false)
}

/* Lists the specified file properties */
func (c *Client) ListFile(fileAndPath string) (*Resources.Resource, error) {
	return c.list(c.path.ToCurrentDir(fileAndPath), true)
}

/* Uses one of the supported features to list a container's resources or the named resource's facts */
func (c *Client) list(path string, isFile bool) (res *Resources.Resource, err error) {
	if !c.InPassiveMode() {
		_, err = c.PassiveMode()
		defer c.RestoreConnections();
	}

	if err == nil {
		if !isFile {
			/* Container listing */
			if c.features.Supports("MLSD") {
				res, err = c.Commands.MLSD(path)

				if !c.Commands.LastIsImplemented() {
					/* MLSD not supported, remove the feature from expected support and fallback on LIST */
					c.features.RemoveFeature("MLSD")
					return c.list(path, isFile)
				}
			} else if c.features.Supports("LIST") {
				res, err = c.Commands.LIST(path)
			}
		} else {
			/* Single resource listing */
			if c.features.Supports("MLST") {
				res, err = c.Commands.MLST(path)

				if !c.Commands.LastIsImplemented() {
					/* MLST not supported, remove the feature from expected support and fallback on LIST */
					c.features.RemoveFeature("MLST")
					return c.list(path, isFile)
				}
			} else if c.features.Supports("LIST") {
				res, err = c.Commands.LIST(path)
			}
		}
	}

	if err == nil {
		c.Resources = res
		fmt.Println(res.GetContentByName("mirror"))
	}

	return
}

/* Log's in with client registered credentials (USER, PASS sequence) */
func (c *Client) LogIn(credentials *Credentials.Credentials) (ok bool, err error) {
	var modified int = -1

	if credentials == nil {
		/* Fallback on existing credentials - if none encoded in the URL this will mean an anonymous login */
		credentials = c.credentials
		modified = 2
	}

	/* Different credentials from client initialization registered ones */
	if modified != 2 {
		if credentials.Username() != c.credentials.Username() || c.credentials.Password() != credentials.Password() {
			/* Keep track of current credentials */
			c.credentials = credentials
		}

		if c.settings.Get(OPT_LoggedIn).Is(true) {
			/* Reset the login status, reinitializing the connection */
			if ok, err = c.Reinitialize(); !ok {
				/* Unable to reinitialize connection to change authentication data */
				return
			}
		}
	}

	/* Log in */
	if c.settings.Get(OPT_LoggedIn).Is(false) {
		_, command := c.requester.Sequence(
			ClientCommands.NewCommand("user", credentials.Username(), Status.UserNameOk),
			ClientCommands.NewCommand("pass", credentials.Password(), Status.UserLoggedIn, Status.AccountForLogin),
		)

		ok, err = command.Success(), command.LastError()

		if ok {
			/* If the server requires an account, send the account information to the server */
			if c.Commands.LastStatus() == Status.AccountForLogin {
				/* Provide the user account to the server */
				if ok, err = c.account(c.settings.Get(OPT_Account).ToString(), false, true); !ok {
					/* Unable to set the specified account */
					return
				}
			}

			if ok {
				/* User account specified/not required */
				c.settings.Get(OPT_LoggedIn).Set(true)
			}
		}
	}

	return
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

/* Connection reinitialization method (uses REIN) */
func (c *Client) Reinitialize() (ok bool, err error) {
	if c.features.Supports("REIN") {
		/* Connection reinitialization attempt */
		if ok, err = c.Commands.REIN(); !ok {
			if !c.Commands.LastIsImplemented() {
				/* Differentiate between common errors and lack of server support (and remember it) */
				err = ERR_ReinNotImplemented
				c.features.RemoveFeature("REIN")
			}
		}
	} else {
		err = ERR_ReinNotImplemented
	}

	/* Reinitialization completed with success */
	if ok {
		/* Restart all affected connection settings */
		c.settings.Get(OPT_LoggedIn).Reset()
		c.settings.Get(OPT_AccountEnabled).Reset()
	}

	return
}

/* Restores the client connection settings to the defaults */
func (c *Client) RestoreConnections() {
	c.settings.Get(OPT_ExtendedPassive).Reset()
	c.settings.Get(OPT_PassiveMode).Reset()
}

/* Makes the client ignore existing files with the same name */
func (c *Client) SetDownloadRuleIgnore() {
	c.settings.Get(OPT_DownloadOverlap).Reset()
}

/* Makes the client overwrite existing files with the same name */
func (c *Client) SetDownloadRuleOverwrite() {
	c.settings.Get(OPT_DownloadOverlap).Set(DO_OverWrite)
}

/* Makes the client create a new copy of the existing files with the same name */
func (c *Client) SetDownloadRuleCreateCopy() {
	c.settings.Get(OPT_DownloadOverlap).Set(DO_CreateNew)
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
	var pathManager *PathManager.PathManager

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

		/* Force the PathManager to use UNIX like path separators. */
		if pathManager, err = PathManager.NewUnixPathManagerAt(RootDir); err != nil {
			return
		}

		client = &Client{commands, requester, credentials, pathManager, Settings.NewSettings(
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
			Settings.NewOption(OPT_DownloadOverlap, DO_IgnoreExisting),
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

/* Sets the account information */
func (c *Client) account(accountInfo string, afterREIN bool, fromLogIn bool) (ok bool, err error) {
	var validAccInfo bool

	/* Check specified account information validity (TODO: refine this step) */
	if accountInfo == EmptyString {
		validAccInfo = false
	}

	if c.settings.Get(OPT_AccountEnabled).Is(true) {
		if c.settings.Get(OPT_Account).Is(accountInfo) {
			/* Same account, return */
			return true, err
		} else if validAccInfo {
			/* Connection reinitialization required to change account information */
			if ok, err = c.Reinitialize(); !ok {
				/* Unable to reinitialize the current control connection */
				return
			} else {
				/* Connection reinitialized, log in back again and provide specified account info */
				return c.account(accountInfo, true, false)
			}
		} else {
			return ok, ERR_LoginAccountRequired
		}
	} else if validAccInfo {
		/* Use the current account information as default account information. */
		c.settings.Get(OPT_Account).Set(accountInfo)
	} else {
		/* Invalid account information */
		return ok, ERR_LoginAccountRequired
	}

	/* If the current connection was authenticated and an account information change was requested, log back in */
	if afterREIN {
		return c.LogIn(nil)
	} else if fromLogIn {
		/* Reply to the server with the current account information if requested from the login sequence */
		ok, err = c.Commands.ACCT(accountInfo)

		if ok {
			/* Mark the current account information as being registered */
			c.settings.Get(OPT_AccountEnabled).Set(true)
		}
	}

	return
}

/* Downloads the specified file */
func (c *Client) downloadFile(file string) (ok bool, err error) {
	downloadBehaviour := c.settings.Get(OPT_DownloadOverlap)
	r := c.Resources.GetContentByName(file)

	/* Download the specified file */
	if r.CanBeRetrieved() {
		if downloadBehaviour.Is(DO_CreateNew) {
			_, err = c.localFM.SelectForWriteNew(file)
		} else if downloadBehaviour.Is(DO_OverWrite) {
			_, err = c.localFM.SelectForWriteTruncate(file)
		} else {
			/* Ignore the current file */
			return true, err
		}

		fmt.Println("Resource selection result", err, downloadBehaviour.Value())

		if err != nil {
			/* Unable to select the local resource */
			err = fmt.Errorf("Download error: Unable to select local resource %s", r.Name)
			return false, err
		}

		fmt.Println("Resource selected")

		/* Put client in passive mode just before downloading */
		if !c.InPassiveMode() {
			_, err = c.PassiveMode()
			defer c.RestoreConnections();
		}

		fmt.Println("Pasive mode ok")

		/* Set representation type */
		if r.IsBinary() {
			c.RepresentationType(ClientCommands.TYPE_Image, nil)
			fmt.Println("Representation type selected", ClientCommands.TYPE_Image)
		} else {
			c.RepresentationType(ClientCommands.TYPE_Ascii, ClientCommands.FMTCTRL_NonPrint)
			fmt.Println("Representation type selected", ClientCommands.TYPE_Ascii)
		}

		/* Only download files with a size greater then 0 */
		if fileRes := c.Resources.GetContentByName(file); fileRes.SizeInkB() > 0 {
			fmt.Println("File resource size: ", fileRes.Size, fileRes, file, "in kb", fileRes.SizeInkB(), "in mb", fileRes.SizeInMB())
			if _, err = c.Commands.RETR(file, c.localFM.GetSelection()); err != nil {
				err = fmt.Errorf("Download error: Unable to RETR file %s. Original error: %s", r.Name, err)
				return
			}
		}

		if err = c.localFM.SelectionClear(); err != nil {
			err = fmt.Errorf("Download error: Unable to close local resource %s", r.Name)
			return
		}
	} else {
		err = ERR_NonRetrievable
	}

	if err == nil {
		ok = true
	}

	return
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
