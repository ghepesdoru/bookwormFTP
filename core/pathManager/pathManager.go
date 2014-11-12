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
	d, f := p.Split(currentDir)
	if f == EmptyString {
		currentDir = p.Clean(d)
	}

	if p.IsAbs(currentDir) {
		/* Make the current directory relative */
		currentDir, err = p.Rel(p.rootDir, currentDir)

		if err != nil {
			return false, ERR_InvalidPath
		}

		/* Replace the current directory for absolute paths */
		p.currentDir = currentDir
	} else {
		currentDir = p.Clean(currentDir)
		p.currentDir = p.Clean(p.currentDir)
		sep := "."
		pSep := p.GetSeparator()
		sep1 := sep + pSep

		if currentDir == sep || sep1 == currentDir {
			currentDir = EmptyString
		}

		if p.currentDir == sep || sep1 == p.currentDir {
			p.currentDir = EmptyString
		}

		if p.currentDir != EmptyString {
			/* Build the current path based on the existing current dir */
			p.currentDir = p.currentDir + p.Clean(currentDir) + pSep
		} else {
			/* Build the current path based on the existing current dir */
			p.currentDir = p.Clean(currentDir) + pSep
		}
	}

	return true, err
}

/* Changes the root directory resetting the current directory */
func (p *PathManager) ChangeRoot(rootDir string) (ok bool, err error) {
	rootDir, _ = p.Split(rootDir)
	rootDir = p.Clean(rootDir)
	if !p.IsAbs(rootDir) {
		/* Consider the specified root directory as being relative to the current working directory */
		rootDir, err = p.Abs(rootDir)

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
	d, f := p.Split(path)
	if p._wrapperRequired() {
		if f == EmptyString {
			if d == UNIXSeparatorString {
				return d
			} else {
				return Path.Clean(d) + UNIXSeparatorString
			}
		} else {
			return Path.Clean(path)
		}
	} else {
		if f == EmptyString {
			return FilePath.Clean(d) + string(FilePath.Separator)
		} else {
			return FilePath.Clean(path)
		}
	}
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

/* Extracts the parent directory's path from the specified path */
func (p *PathManager) ExtractParentDir(path string) string {
	parts := p.SplitDirList(path)
	length := len(parts)

	if length > 0 {
		if length > 1 {
			/* Exclude any file from the path */
			if !p.IsDir(parts[length - 1]) {
				parts = parts[:length - 2]

				return p.ExtractParentDir(p.Join(parts...))
			}

			return p.Join(parts[:length - 2]...)
		}
	}

	return EmptyString
}

/* Extracts the current directory from the path, leaving a relative path without any relationship markers */
func (p *PathManager) ExtractSubPath(path string) []string {
	if p.InCurrentDir(path) {
		l1 := len(p.GetCurrentDir())
		l2 := len(path)

		/* If it's the same directory ignore the case, we have nothing to return */
		if l1 != l2 {
			return p.SplitDirList(path[l2:])
		}
	}

	return []string{}
}

/* Normalizes the current directory. */
func (p *PathManager) _getCurrentDir() string {
	if p.currentDir == EmptyString {
		return EmptyString
	} else {
		return p.Clean(p.currentDir)
	}
}

/* Current directory path getter. It will always return an absolute path. */
func (p *PathManager) GetCurrentDir() string {
	return p.Clean(p.rootDir) + p._getCurrentDir()
}

/* Root directory getter. */
func (p *PathManager) GetRootDir() string {
	return p.rootDir
}

/* File path separator getter */
func (p *PathManager) GetSeparator() string {
	if p._wrapperRequired() {
		return UNIXSeparatorString
	} else {
		return string(FilePath.Separator)
	}
}

/* Wrapper around FilePath.IsAbs() that takes OS emulation into consideration */
func (p *PathManager) IsAbs(path string) bool {
	if p._wrapperRequired() {
		/* Threat '/' as root directory. */
		return Path.IsAbs(path)
	}

	return FilePath.IsAbs(path)
}

/* Checks if the current path is a directory path */
func (p *PathManager) IsDir(path string) bool {
	return string(path[len(path) - 1]) == p.GetSeparator()
}

/* Checks if the specified path is the current directory */
func (p *PathManager) IsCurrentDir(path string) bool {
	return path == p.GetCurrentDir()
}

/* Checks if the specified path is in the current directory */
func (p *PathManager) InCurrentDir(path string) bool {
	if p.IsCurrentDir(path) {
		return true
	}

	curr := p.GetCurrentDir()

	if !p.IsAbs(path) {
		/* Relative path. Try a concatenation, if the resulting dir is not the current directory, dump the path as a negative */
		d, _ := p.Split(p.Join(curr, path))

		if d == curr {
			return true
		}
	} else {
		/* Absolute path. Check if the specified path overlaps the specified path entirely */
		l1 := len(curr)
		l2 := len(path)

		if l1 < l2 {
			/* The current path has to be either a subdirectory or a non related path at this point */
			if path[:l1 - 1] == curr {
				/* This is a path descending from the current path */
				return true
			}
		}
	}

	return false
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
		db, _ := p.Split(basepath)
		dt, ft := p.Split(targpath)

		basepath = p.Clean(db)
		targpath = p.Clean(dt)

		if basepath == targpath {
			return ".", err
		}

		if basepath == UNIXSeparatorString && p.IsAbs(targpath) {
			return "./" + targpath[1:], err
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
				return "./" + strings.Join(tp, UNIXSeparatorString) + UNIXSeparatorString + p.Clean(ft), nil
			} else if matches < blen {
				/* Sibling of the current directory or it's parents */
				a := []string{"."}
				for i := 0; i < (blen - matches); i += 1 {
					a = append(a, "..")
				}

				return strings.Join(a, UNIXSeparatorString), nil
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

/* Breaks down a path in a list of subdirectories */
func (p *PathManager) SplitDirList(dir string) []string {
	if p._wrapperRequired() {
		return strings.Split(p.Clean(p.SplitDir(dir)), UNIXSeparatorString)
	}

	return strings.Split(p.Clean(p.SplitDir(dir)), string(FilePath.Separator))
}

/* Split the specified path to extract the file */
func (p *PathManager) SplitFile(path string) string {
	_, path = p.Split(path)
	return path
}

/* Attaches the specified file name to the current path */
func (p *PathManager) ToCurrentDir(fileName string) string {
	var dir, file string
	var err error

	if fileName == p.GetCurrentDir() {
		return fileName
	}

	dir, file = p.Split(fileName)

	if p.IsAbs(dir) {
		dir, err = p.Rel(p.GetCurrentDir(), dir)

		if err != nil {
			/* There is no relationship between the specified path and the current directory, just add the file to the current directory */
			return p.GetCurrentDir() + file
		}
	}

	/* Remove local path from directory concatenation */
	if dir == ("." + p.GetSeparator()) {
		dir = ""
	}

	dir = p.GetCurrentDir() + dir

	if file != EmptyString {
		return p.Join(dir, file)
	} else {
		return p.Clean(dir)
	}
}

/* Forces the Path Manager to use only / as path separator, and normalize all paths to be UNIX like. */
func (p *PathManager) UnixOnlyMode(enable bool) {
	p.unixOnly = enable
}
