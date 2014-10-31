package fileManager

import (
	BaseParser "github.com/ghepesdoru/bookwormFTP/core/parsers/base"
	"fmt"
	"os"
	"path"
	"strconv"
)

/* Definition of selection type */
type selectionType struct {
	open_flag		int
	permissions		os.FileMode
}

type selection int

const (
	/* Selection types naming constants */
	SELECT_ReadOnly	selection = iota
	SELECT_WriteOnly
	SELECT_ReadWrite
	SELECT_Append
	SELECT_Truncate
	SELECT_CreateNew
)

var (
	ERR_InvalidPath 			= fmt.Errorf("Invalid root directory path. Please consider using an absolute path.")
	ERR_InvalidSelection 		= fmt.Errorf("Invalid directory selection. Please consider selecting a file.")

	/* Selection types definitions */
	SelectionTypes				= map[selection]selectionType {
		SELECT_ReadOnly:		selectionType{os.O_RDONLY,	os.ModePerm},
		SELECT_WriteOnly:		selectionType{os.O_WRONLY, 	os.ModePerm},
		SELECT_ReadWrite:		selectionType{os.O_RDWR, 	os.ModePerm},
		SELECT_Append:			selectionType{os.O_APPEND,	os.ModePerm},
		SELECT_Truncate:		selectionType{os.O_TRUNC,	os.ModePerm},
		SELECT_CreateNew:		selectionType{os.O_WRONLY,	os.ModePerm},
	}
)

/* Define the FileManager type */
type FileManager struct {
	rootDir		string
	cwd			*os.File
	listing		[]os.FileInfo
	focus		*os.File
}

/* Instantiates a new FileManager in the current process's working dir */
func NewFileManager() (*FileManager, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	pwd = path.Clean(pwd) + "/"
	return NewFileManagerAt(pwd)
}

/* Instantiates a new FileManager in the specified directory */
func NewFileManagerAt(rootDir string) (fm *FileManager, err error) {
	fm = &FileManager{rootDir, nil, []os.FileInfo{}, nil}
	_, err = fm.ChangeDir(rootDir)

	return
}

/* Changes the current directory */
func (fm *FileManager) ChangeDir(dir string) (ok bool, err error) {
	var f os.File
	var listing []os.FileInfo

	dir = path.Clean(dir) + "/"

	if !path.IsAbs(dir) {
		return ok, ERR_InvalidPath
	}

	if f, err := os.Open(dir); err == nil {
		listing, err = f.Readdir(-1)
	}

	if err == nil {
		fm.cwd = &f
		fm.listing = listing
		fm.rootDir = dir
		f.Close()
		ok = true
	}

	return
}

/* Changes the path relative to the current parent directory */
func (fm *FileManager) ChangeDirRelative(localPath string) (ok bool, err error) {
	return fm.ChangeDir(fm.rootDir + localPath)
}

/* Checks if the current directory contains the specified resource taking type into consideration */
func (fm *FileManager) contains(name string, isDir bool) bool {
	for _, f := range fm.listing {
		if f.Name() == name && f.IsDir() == isDir {
			return true
		}
	}

	return false
}

/* Checks if the current directory contains the specified subdirectory */
func (fm *FileManager) ContainsDir(dir string) bool {
	return fm.contains(dir, true)
}

/* Checks the current directory listing for the specified file existence */
func (fm *FileManager) ContainsFile(fileName string) bool {
	return fm.contains(fileName, false)
}

/* Creates a new file */
func (fm *FileManager) CreateFile(fileName string) (ok bool, err error) {
	var f *os.File
	var fs os.FileInfo

	f, err  = os.Create(fileName)

	if err == nil {
		if fs, err = f.Stat(); err == nil {
			fm.listing = append(fm.listing, fs)
		} else {
			ok, err = fm.ChangeDir(fm.cwd.Name())
		}

		f.Close()
	}

	return err == nil, err
}

/* Get the file currently in focus */
func (fm *FileManager) GetSelection() *os.File {
	if fm.focus != nil {
		return fm.focus
	}

	return nil
}

/* Lists the contents of the current directory */
func (fm *FileManager) List() []string {
	var list []string
	for _, f := range fm.listing {
		list = append(list, fmt.Sprintf("size=%d;modify=%s;perm=%s;type=%s %s", f.Size(), BaseParser.ToTimeVal(f.ModTime()), fm.listFilePerm(f.Mode()), fm.listFileType(f.IsDir()), f.Name()))
	}

	return list
}

/* List() method helper */
func (fm *FileManager) listFilePerm(m os.FileMode) string {
	return m.String()
}

/* List() method helper */
func (fm *FileManager) listFileType(isDir bool) string {
	if isDir {
		return "dir"
	}

	return "file"
}

/* Create a new directory */
func (fm *FileManager) MakeDir(dir string) (ok bool, err error) {
	err = os.Mkdir(dir, os.ModePerm)

	if err == nil {
		if s, e := os.Stat(fm.rootDir + dir); e == nil {
			fm.listing = append(fm.listing, s)
		} else {
			err = e
		}
	}

	return err == nil, err
}

/* File structure getter, can be used in cases where a reader/writer is required */
func (fm *FileManager) Select(fileName string, se selection) (f *os.File, err error) {
	return fm._select(fileName, se, 0)
}

/* File structure getter with support for incremental file name creation. */
func (fm *FileManager) _select(fileName string, se selection, increment int) (f *os.File, err error) {
	var s os.FileInfo
	var selection selectionType
	selection, _ = SelectionTypes[se]

	if fm.focus != nil {
		err = fm.SelectionClear()
	}

	if err != nil {
		return
	}

	if fm.ContainsFile(fileName) {
		if se == SELECT_CreateNew {
			/* Create a new file if a file with the specified name exists */
			ext := path.Ext(fileName)
			l := len(fileName)
			if increment == 0 {
				fileName = fileName[:l - len(ext)] + "_" + strconv.Itoa(increment) + ext
			} else {
				fileName = fileName[:l - (len(ext) + 2)] + "_" + strconv.Itoa(increment) + ext
			}
			return fm._select(fileName, se, increment + 1)
		} else {
			f, err = os.OpenFile(path.Join(fm.rootDir, fileName), selection.open_flag, selection.permissions)

			if err == nil {
				s, err = f.Stat()
				if !s.IsDir() {
					fm.focus = f
				} else {
					f.Close()
					err = ERR_InvalidSelection
				}
			}
		}
	} else if se != SELECT_ReadOnly && se != SELECT_Append && se != SELECT_Truncate {
		/* Create the new file */
		_, err = fm.CreateFile(path.Join(fm.rootDir, fileName))

		if err != nil {
			return
		}

		/* Make sure not to continue generating new file names in a loop */
		if se == SELECT_CreateNew {
			se = SELECT_WriteOnly
		}

		/* A file with a new name was created, recursively open it for write */
		return fm._select(fileName, se, increment)
	}

	return
}

/* Specialized selection: Reading only */
func (fm *FileManager) SelectForRead(fileName string) (f *os.File, err error) {
	return fm.Select(fileName, SELECT_ReadOnly)
}

/* Specialized selection: Reading and writing */
func (fm *FileManager) SelectForReadWrite(fileName string) (f *os.File, err error) {
	return fm.Select(fileName, SELECT_ReadWrite)
}

/* Specialized selection: Write only */
func (fm *FileManager) SelectForWrite(fileName string) (f *os.File, err error) {
	return fm.Select(fileName, SELECT_WriteOnly)
}

/* Specialized selection: Write after truncate */
func (fm *FileManager) SelectForWriteTruncate(fileName string) (f *os.File, err error) {
	return fm.Select(fileName, SELECT_Truncate)
}

/* Specialized selection: Write after creating a new unique file */
func (fm *FileManager) SelectForWriteNew(fileName string) (f *os.File, err error) {
	return fm.Select(fileName, SELECT_CreateNew)
}

/* Specialized selection: Reading only */
func (fm *FileManager) SelectForAppend(fileName string) (f *os.File, err error) {
	return fm.Select(fileName, SELECT_Append)
}

/* Closes the currently opened file. */
func (fm *FileManager) SelectionClear() (err error) {
	if fm.focus != nil {
		if err = fm.focus.Sync(); err == nil {
			err = fm.focus.Close()
		}
	}

	return
}
