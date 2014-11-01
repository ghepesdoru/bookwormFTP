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

	fmt.Println(fmt.Sprintf("FileName: %s, dir: %s, file: %s", fileName, dir, file))

	if ok, err = c.ChangeDir(dir); ok {
		/* Use the last subdirectory as container for the downloaded content */
		if len(file) == 0 {
			/* Download the entire current directory */
			return c.downloadDir(c.path.SplitDir(dir), false)
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
	if !c.localFM.ContainsDir(currentDir) {
		fmt.Println("Non existing directory")
		fmt.Println("List", c.localFM.List())
		fmt.Println("Args", currentDir, changePath)

		/* Create a new directory */
		if ok, err = c.localFM.MakeDir(currentDir); !ok {
			err = fmt.Errorf("Download error: Unable to create local directory %s. Original error: %s", currentDir, err)
			return
		}
	}

	/* Change to the existing directory */
	if ok, err = c.localFM.ChangeDir(currentDir); !ok {
		fmt.Println("Existing directory selection")
		fmt.Println(currentDir, ok, err)
		fmt.Println(c.localFM.List())
		err = fmt.Errorf("Download error: Unable to change path to local directory %s", currentDir)
		return
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
			if !f.IsChild() {
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
		if pathManager, err = PathManager.NewPathManagerAt(RootDir); err != nil {
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

		/* Force the PathManager to use UNIX like path separators. */
		client.path.UnixOnlyMode(true)
		client.path.ChangeRoot(RootDir)

		/* Enable debugging */
		if client.settings.Get(OPT_DebugMode).Is(true) {
			requester.Logger = Logger.NewSimpleLogger()
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
		} else {
			c.RepresentationType(ClientCommands.TYPE_Ascii, ClientCommands.FMTCTRL_NonPrint)
		}

		fmt.Println("Representation type selected")

		if _, err = c.Commands.RETR(file, c.localFM.GetSelection()); err != nil {
			err = fmt.Errorf("Download error: Unable to RETR file %s. Original error: %s", r.Name, err)
			return
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
