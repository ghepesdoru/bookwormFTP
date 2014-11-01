package pathManager

import(
	BaseParser "github.com/ghepesdoru/bookwormFTP/core/parsers/base"
	Path "path/filepath"
	"fmt"
	"os"
)

const (
	EmptyString = ""
)

var (
	ERR_InvalidRootPath = fmt.Errorf("Invalid root directory path. Please consider using an absolute path.")
	ERR_InvalidPath = fmt.Errorf("Invalid specified directory path. Please consider using an absolute path.")

	Underline = "_"[0]
)

	/* Definition of the PathManager type */
type PathManager struct {
	rootDir		string
	currentDir	string
}

/* Instantiate a new path manager in the current working directory */
func NewPathManager() (*PathManager, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	return NewPathManagerAt(pwd)
}

/* Instantiate a new path manager at the specified location */
func NewPathManagerAt(rootDir string) (p *PathManager, err error) {
	var ok bool

	p = &PathManager{EmptyString, EmptyString}

	ok, err = p.ChangeRoot(rootDir)
	if !ok {
		return nil, err
	}

	return
}

/* Change the current directory */
func (p *PathManager) ChangeCurrentDir(currentDir string) (ok bool, err error) {
	if Path.IsAbs(currentDir) {
		/* Make the current directory relative */
		currentDir, err = Path.Rel(p.rootDir, currentDir)

		if err != nil {
			return false, ERR_InvalidPath
		}
	}

	p.currentDir = currentDir

	return true, err
}

/* Changes the root directory resetting the current directory */
func (p *PathManager) ChangeRoot(rootDir string) (ok bool, err error) {
	if !Path.IsAbs(rootDir) {
		/* Consider the specified root directory as being relative to the current working directory */
		rootDir, err = Path.Abs(rootDir)

		if err != nil {
			return ok, ERR_InvalidRootPath
		}
	} else {
		rootDir = Path.Clean(rootDir)
	}

	p.rootDir = rootDir
	p.currentDir = EmptyString

	return true, err
}

/* Current directory path getter. It will always return an absolute path. */
func (p *PathManager) GetCurrentDir() string {
	return Path.Join(p.rootDir, p.currentDir)
}

/* Root directory getter. */
func (p *PathManager) GetRootDir() string {
	return p.rootDir
}

/* Attaches the specified file name to the current path */
func (p *PathManager) ToCurrentDir(fileName string) string {
	dir, file := Path.Split(fileName)
	dir, _ = Path.Rel(p.GetCurrentDir(), dir)
	dir = Path.Join(p.GetCurrentDir(), dir)
	return Path.Join(dir, file)
}

/* Iterate the specified file name */
func (p *PathManager) IterateFileName(file string) string {
	var length, increment int

	_, file = Path.Split(file)
	ext := Path.Ext(file)
	file = file[:len(file) - len(ext)]
	length = len(file)

	for i := (length - 1); i > -1; i -= 1 {
		if file[i] == Underline {
			increment = BaseParser.ToInt([]byte(file[i:]))
			fmt.Println("Increment", increment, file[i:], file[:i], file[i:])

			if increment > -1 {
				file = file[:i]
			}

			break
		}
	}

	if increment == -1 {
		increment = 0
	}

	return fmt.Sprintf("%s_%d%s", file, increment, ext)
}
