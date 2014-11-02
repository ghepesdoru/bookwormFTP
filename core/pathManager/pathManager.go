package pathManager

import(
	BaseParser 	"github.com/ghepesdoru/bookwormFTP/core/parsers/base"
	FilePath 	"path/filepath"
	Path		"path"
	"fmt"
	"os"
	"strings"
)

const (
	EmptyString = ""
	UNIXSeparatorString = "/"
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
	unixOnly	bool
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
	return newPathManagerAt(rootDir, false)
}

func NewUnixPathManagerAt(rootDir string) (p *PathManager, err error) {
	return newPathManagerAt(rootDir, true)
}

/* Instantiates a new Path manager in unix mode/current mode at specified location */
func newPathManagerAt(rootDir string, unixMode bool) (p *PathManager, err error) {
	var ok bool

	p = &PathManager{EmptyString, EmptyString, unixMode}

	ok, err = p.ChangeRoot(rootDir)
	if !ok {
		return nil, err
	}

	return
}

/* Checks if the PATH SEPARATOR is '\\' */
func (p *PathManager) _isWindows() bool {
	return FilePath.Separator == '\\'
}

/* Checks if wrapper's alternative code has to be employed */
func (p *PathManager) _wrapperRequired() bool {
	return p.unixOnly && p._isWindows()
}

/* Wrapper around FilePath.Abs() taking OS emulation into account */
func (p *PathManager) Abs(path string) (string, error) {
	if p._wrapperRequired() {
		if p.IsAbs(path) {
			return p.Clean(path), nil
		}

		return p.Join(p.rootDir, path), nil
	}

	fmt.Println("Abs executes without entering the wrapper thing")

	return FilePath.Abs(path)
}

/* Wrapper around FilePath.Base() taking OS emulation into account */
func (p *PathManager) Base(path string) string {
	if p._wrapperRequired() {
		return Path.Base(path)
	}

	return FilePath.Base(path)
}

/* Change the current directory */
func (p *PathManager) ChangeCurrentDir(currentDir string) (ok bool, err error) {
	if p.IsAbs(currentDir) {
		/* Make the current directory relative */
		currentDir, err = p.Rel(p.rootDir, currentDir)

		if err != nil {
			return false, ERR_InvalidPath
		}
	}

	p.currentDir = currentDir

	return true, err
}

/* Changes the root directory resetting the current directory */
func (p *PathManager) ChangeRoot(rootDir string) (ok bool, err error) {
	fmt.Println("current root", p.rootDir, "new root:", rootDir)
	if !p.IsAbs(rootDir) {
		/* Consider the specified root directory as being relative to the current working directory */
		rootDir, err = p.Abs(rootDir)

		fmt.Println("Root after abs", rootDir)

		if err != nil {
			return ok, ERR_InvalidRootPath
		}
	} else {
		rootDir = p.Clean(rootDir)
	}

	p.rootDir = rootDir
	p.currentDir = EmptyString

	return true, err
}

/* Cleans the path. Takes OS emulation into consideration */
func (p *PathManager) Clean(path string) string {
	if p._wrapperRequired() {
		return Path.Clean(path)
	}

	return FilePath.Clean(path)
}

/* Wrapper around FilePath.Dir() that takes OS emulation into consideration */
func (p *PathManager) Dir(path string) string {
	if p._wrapperRequired() {
		return Path.Dir(path)
	}

	return FilePath.Dir(path)
}

/* Wrapper around FilePath.Ext() that takes OS emulation into consideration */
func (p *PathManager) Ext(path string) string {
	if p._wrapperRequired() {
		return Path.Ext(path)
	}

	return FilePath.Ext(path)
}

/* Current directory path getter. It will always return an absolute path. */
func (p *PathManager) GetCurrentDir() string {
	return p.Join(p.rootDir, p.currentDir)
}

/* Root directory getter. */
func (p *PathManager) GetRootDir() string {
	return p.rootDir
}

/* Wrapper around FilePath.IsAbs() that takes OS emulation into consideration */
func (p *PathManager) IsAbs(path string) bool {
	if p._wrapperRequired() {
		/* Threat '/' as root directory. */
		return Path.IsAbs(path)
	}

	return FilePath.IsAbs(path)
}

/* Checks if the current dir is the root dir */
func (p *PathManager) InRootDir() bool {
	return p.rootDir == p.currentDir
}

/* Iterate the specified file name */
func (p *PathManager) IterateFileName(file string) string {
	var length, increment int

	_, file = p.Split(file)
	ext := p.Ext(file)
	file = file[:len(file) - len(ext)]
	length = len(file)

	for i := (length - 1); i > -1; i -= 1 {
		if file[i] == Underline {
			increment = BaseParser.ToInt([]byte(file[i:]))

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

/* Wrapper around FilePath.Join() that takes OS emulation into consideration */
func (p *PathManager) Join(elem ...string) string {
	if p._wrapperRequired() {
		return Path.Join(elem...)
	}

	return FilePath.Join(elem...)
}

/* Wrapper around filepath.Rel() that takes OS emulation into consideration */
func (p *PathManager) Rel(basepath, targpath string) (path string, err error) {
	if p._wrapperRequired() {
		basepath = p.Clean(basepath)
		targpath = p.Clean(targpath)

		if basepath == targpath {
			return ".", nil
		}

		bp := strings.Split(basepath, UNIXSeparatorString)
		tp := strings.Split(targpath, UNIXSeparatorString)
		var matches, matchStart, tlen int = 0, -1, len(tp) - 1

		/* Get last matching part */
		for i, b := range bp {
			if i > tlen || tp[i] != b {
				/* End of matching parts */
				break
			} else {
				matches += 1
				if matchStart == -1 {
					matchStart = i
				}
			}
		}

		if matches > 0 {
			blen := len(bp)

			if matches == blen {
				/* Subdirectory of the base path */
				tp = tp[matches:]
				fmt.Println("Gets here")
				return "./" + strings.Join(tp, UNIXSeparatorString), nil
			} else if matches < blen {
				/* Sibling of the current directory or it's parents */
				a := []string{"."}
				for i := 0; i < (blen - matches); i += 1 {
					a = append(a, "..")
				}

				return strings.Join(a, "/"), nil
			}
		} else {
			return EmptyString, fmt.Errorf("Rel: can't make %s relative to %s.", targpath, basepath)
		}
	}

	return FilePath.Rel(basepath, targpath)
}

/* Wrapper around path.Split() that takes OS emulation into consideration */
func (p *PathManager) Split(path string) (string, string) {
	if p._wrapperRequired() {
		return Path.Split(path)
	}

	return FilePath.Split(path)
}

/* Split the specified path to extract the directory */
func (p *PathManager) SplitDir(path string) string {
	path, _ = p.Split(path)
	return path
}

/* Split the specified path to extract the file */
func (p *PathManager) SplitFile(path string) string {
	_, path = p.Split(path)
	return path
}

/* Attaches the specified file name to the current path */
func (p *PathManager) ToCurrentDir(fileName string) string {
	if fileName == p.GetCurrentDir() {
		return fileName
	}

	dir, file := p.Split(fileName)
	dir, _ = p.Rel(p.GetCurrentDir(), dir)
	dir = p.Join(p.GetCurrentDir(), dir)
	return p.Join(dir, file)
}

/* Forces the Path Manager to use only / as path separator, and normalize all paths to be UNIX like. */
func (p *PathManager) UnixOnlyMode(enable bool) {
	p.unixOnly = enable

	fmt.Println("Path rel by FilePath: ")
	fmt.Println(FilePath.Abs("a/b/c/d///"))
	fmt.Println("Path rel by wrapper: ")
	fmt.Println(p.Abs("a/b/c/d///"))
}
