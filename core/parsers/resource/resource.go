package resource

import (
	Access "github.com/ghepesdoru/bookwormFTP/core/access"
	BaseParser "github.com/ghepesdoru/bookwormFTP/core/parsers/base"
	"fmt"
	"regexp"
	"time"
)

/* Resource type */
type ResourceType int
const(
	TYPE_File ResourceType = iota
	TYPE_Dir
	TYPE_CDir
	TYPE_PDir
	TYPE_Other
)

type MIMEType int
const (
	MIME_Unknown MIMEType = iota
	MIME_Text
	MIME_Binary
)

const (
	Comma = 59
	Equal = 61
	EmptyString = ""
	DateFormat = "Jan 2 2008"
)

var (
	ExtractListParts = regexp.MustCompile(`(?:^[ldrwx-]*[[:blank:]]*[[:digit:]]*[[:blank:]]*[[:alpha:]]*[[:blank:]]*[[:alpha:]]*[[:blank:]]*)([0-9]*)(?: *)([A-Za-z]*.[0-9]*(?: *)[0-9]*)(?: *)([A-Za-z*_.-]+)*(?:.*)`)
	UnknownTime = time.Unix(0, 0)
	StringToTYPEMap = map[string]ResourceType {
		"file": TYPE_File,	"dir": TYPE_Dir,	"cdir": TYPE_CDir,	"pdir": TYPE_PDir,
	}
)

/* File type definition */
type Resource struct {
	Name		string
	Size		int
	Modify		*time.Time
	Create		*time.Time
	Type		ResourceType
	Unique		string
	Permissions	*Access.AccessRights
	Language	string
	MIME		MIMEType
	Charset		string
	Parent		*Resource
	Content		[]*Resource
}

/* Instantiates a new resource */
func NewResource(name string,size int, modify *time.Time, create *time.Time, rType ResourceType, unique string, access *Access.AccessRights, lang string, mime MIMEType, charset string) *Resource {
	return &Resource{name, size, modify, create, rType, unique, access, lang, mime, charset, nil, nil}
}

/* Extracts the resource from a MLSx formatted list */
func FromMLSxList(list []byte) (res *Resource, err error) {
	var lines [][]byte
	var r *Resource
	var containerFound bool

//	modify=20130726115449;perm=fle;type=cdir;unique=13U31245A;UNIX.group=50;UNIX.mode=0755;UNIX.owner=14; .
	lines = BaseParser.SplitLines(list)

	for _, l := range lines {
		r, err = parseMLSx(l)

		if err != nil {
			break
		}

		if !containerFound {
			if r.IsCurrentDir() {
				containerFound = true
				res = r
			} else {
				err = fmt.Errorf("No container resource found.")
			}
		} else {
			if r.IsParentDir() {
				/* Attach the parent of the current resource */
				res.Parent = r
			} else {
				/* Attach the resource to it's container */
				r.Parent = res
				res.Content = append(res.Content, r)
			}
		}
	}

	return
}

/* Extracts the resource from a List formatted list */
func FromList(list []byte) (res *Resource, err error) {
	var lines [][]byte
	var r *Resource
	var length, size int
	var name string
	var modified time.Time
	var resType ResourceType = TYPE_Other
	var mime MIMEType = MIME_Unknown

	/* Define a virtual container for resource functionality uniformity. */
	res = &Resource{".", size, &UnknownTime, &UnknownTime, TYPE_Dir, EmptyString, Access.NewEmptyAccessRights(), EmptyString, mime, EmptyString, nil, nil}
	lines = BaseParser.SplitLines(list)
	for _, l := range lines {
		/* Extract current line's content */
		parts := ExtractListParts.FindAllSubmatch(l, -1)
		for _, matches := range parts {
			length = len(matches)

			if length == 4 {
				size = BaseParser.ToInt(matches[1])
				modified, err = time.Parse(DateFormat, string(matches[2]))
				name = string(matches[3])

				if nil == err {
					/* Extract mime type and normalize resource type using available information */
					mime = determineMIME(name)

					if mime != MIME_Unknown {
						resType = TYPE_File
					} else {
						resType = TYPE_Dir
					}

					r = &Resource{name, size, &modified, &UnknownTime, resType, EmptyString, Access.NewEmptyAccessRights(), EmptyString, mime, EmptyString, nil, nil}
					res.Content = append(res.Content, r)
				}
			}
		}
	}

	return
}

/* Brakes a single MLSx response line into it's component parts and fills a resource with found information */
func parseMLSx (line []byte) (res *Resource, err error) {
	var length int = len(line) - 1
	var start, end, fEnd int = 0, -1, -1
	var name, unique, language, charset string
	var fact, value []byte
	var size int
	var modify, create *time.Time = &UnknownTime, &UnknownTime
	var resType ResourceType = TYPE_Other
	var perms *Access.AccessRights
	var mime MIMEType = MIME_Unknown

	for i, c := range line {
		if c == Comma {
			/* End of resource fact */
			end = i
			fact = BaseParser.ToLower(line[start:fEnd])
			value = line[fEnd+1:end]

			switch string(fact) {
			case "size":
				size = BaseParser.ToInt(value)
			case "modify":
				modify, err = BaseParser.ParseTimeVal(value)
			case "create":
				create, err = BaseParser.ParseTimeVal(value)
			case "type":
				if v, ok := StringToTYPEMap[string(value)]; ok {
					resType = v
				}
			case "unique", "lang", "charset":
				unique = string(value)
			case "perm":
				perms = Access.FromPermString(value)
			}

			start = end + 1
		} else if BaseParser.IsWhitespace(c) {
			/* Start of resource name */
			start = i + 1
			end = length
			name = string(BaseParser.Trim(line[start:end]))
			break
		} else if c == Equal {
			fEnd = i
		}

		if err != nil {
			break
		}
	}

	if resType == TYPE_File && len(name) > 0 {
		ext := BaseParser.SplitOnSeparator([]byte(name), []byte{BaseParser.CONST_Dot})
		length = len(ext)

		if length > 0 {
			mime = determineMIME(string(ext[length - 1]))
		} else {
			mime = determineMIME("")
		}
	}

	if err == nil {
		res = &Resource{name, size, modify, create, resType, unique, perms, language, mime, charset, nil, nil}
	} else {
		/* Debug point */
//		fmt.Println("Resource build error:", err)
	}

	return
}

/* Determine a file's MIME type (reused for data connection configuration) */
func determineMIME(fileExtension string) (mime MIMEType) {
	switch fileExtension {
	case
		"323", "bas", "c", "css", "etx", "h", "htc", "htm",
		"html", "htt", "rtx", "sct", "stm", "tsv", "txt", "uls",
		"vcf", "xml":
		mime = MIME_Text
	case
		"", "*", "acx", "ai", "aif", "aifc", "aiff", "asf", "asr", "asx",
		"au", "avi", "axs", "bcpio", "bin", "bmp", "cat", "cdf",
		"cer", "class", "clp", "cmx", "cod", "cpio", "crd", "crl",
		"crt", "csh", "dcr", "der", "dir", "dll", "dms", "doc", "dot",
		"dvi", "dxr", "eps", "evy", "exe", "fif", "flr", "gif", "gtar",
		"gz", "hdf", "hlp", "hqx", "hta", "ico", "ief", "iii", "img", "ins",
		"iso", "isp", "jfif", "jpe", "jpeg", "jpg", "js", "latex", "lha", "lsf",
		"lsx", "lzh", "m13", "m14", "m3u", "man", "mdb", "me", "mht",
		"mhtml", "mid", "mny", "mov", "movie", "mp2", "mp3", "mpa", "mpe",
		"mpeg", "mpg", "mpp", "mpv2", "ms", "msg", "mvb", "nc", "nws",
		"oda", "p10", "p12", "p7b", "p7c", "p7m", "p7r", "p7s", "pbm",
		"pdf", "pfx", "pgm", "pko", "pma", "pmc", "pml", "pmr", "pmw",
		"pnm", "pot", "ppm", "pps", "ppt", "prf", "ps", "pub", "qt", "ra",
		"ram", "ras", "rgb", "rmi", "roff", "rtf", "scd", "setpay", "setreg",
		"sh", "shar", "sit", "snd", "spc", "spl", "src", "sst", "stl", "sv4cpio",
		"sv4crc", "svg", "swf", "t", "tar", "tcl", "tex", "texi", "texinfo",
		"tgz", "tif", "tiff", "tr", "trm", "ustar", "vrml", "wav", "wcm", "wdb",
		"wks", "wmf", "wps", "wri", "wrl", "wrz", "xaf", "xbm", "xla", "xlc",
		"xlm", "xls", "xlt", "xlw", "xof", "xpm", "xwd", "z", "zip":
		mime = MIME_Binary
	default:
		mime = MIME_Unknown
	}

	return
}

/* Checks if the current resource ca be appended */
func (r *Resource) CanBeAppended() bool {
	return r.IsFile() && r.Permissions.Contains(Access.PERM_Append)
}

/* Checks if the current resource can be extended (new resources added/appended) */
func (r *Resource) CanBeExtended() bool {
	return r.IsDir() && r.Permissions.Contains(Access.PERM_Create)
}

/* Checks if the current resource can be listed by a LIST/NLST/MLSD command */
func (r *Resource) CanBeListed() bool {
	return r.IsDir() && r.Permissions.Contains(Access.PERM_List)
}

/* Checks if the current resource can be navigated by using the CDUP and CWD commands */
func (r *Resource) CanBeNavigated() bool {
	return r.IsDir() && r.Permissions.Contains(Access.PERM_Execute)
}

/* Checks if all contained elements of the current resource can be removed */
func (r *Resource) CanBePurged() bool {
	return r.IsDir() && r.Permissions.Contains(Access.PERM_Purge)
}

/* Checks if the current resource can be renamed */
func (r *Resource) CanBeRenamed() bool {
	return r.Permissions.Contains(Access.PERM_Rename)
}

/* Checks if the current resource can be removed from it's container */
func (r *Resource) CanBeRemoved() bool {
	return r.Permissions.Contains(Access.PERM_Delete)
}

/* Checks if the current resource can be downloaded */
func (r *Resource) CanBeRetrieved() bool {
	return (r.IsFile() || r.Type == TYPE_Other) && r.Permissions.Contains(Access.PERM_Retrievable)
}

/* Checks if the current resource can be stored (STOR) */
func (r *Resource) CanBeStored() bool {
	return r.IsFile() && r.Permissions.Contains(Access.PERM_Storable)
}

/* Checks if the current resource become the parent container of a new dir resource */
func (r *Resource) CanHostNewDirs() bool {
	return r.IsDir() && r.Permissions.Contains(Access.PERM_Make)
}

/* Checks if any contained resource has the specified name */
func (r *Resource) ContainsByName(resourceName string) bool {
	return r.GetContentByName(resourceName) != nil
}

/* Check if the two resources refer to the same resource */
func (r *Resource) Equals(res *Resource) bool {
	return r.Name == res.Name && r.Unique == res.Unique
}

/* Gets the first resource matching the specified name from the current resource's contents */
func (r *Resource) GetContentByName(resourceName string) *Resource {
	for _, res := range r.Content {
		if res.Name == resourceName {
			return res
		}
	}

	return nil
}

/* Gets the first resource matching the specified unique from the current resource's contents */
func (r *Resource) GetContentByUnique(unique string) *Resource {
	for _, res := range r.Content {
		if res.Unique == unique {
			return res
		}
	}

	return nil
}

/* Checks if the current resource is binary */
func (r *Resource) IsBinary() bool {
	return r.MIME == MIME_Binary
}

/* Checks if the current resource represents the child of a parent resource (excludes self and parent dir) */
func (r *Resource) IsChild() bool {
	return r.Type == TYPE_Dir || r.Type == TYPE_File || r.Type == TYPE_Other
}

/* Checks if the current resource is the current container directory */
func (r *Resource) IsCurrentDir() bool {
	return r.Type == TYPE_CDir
}

/* Checks if the current resource of one of the dir types */
func (r *Resource) IsDir() bool {
	return r.Type == TYPE_CDir || r.Type == TYPE_PDir || r.Type == TYPE_Dir
}

/* Checks if the current resource is a file */
func (r *Resource) IsFile() bool {
	return r.Type == TYPE_File
}

/* Checks if the current resource is a the container directory */
func (r *Resource) IsParentDir() bool {
	return r.Type == TYPE_PDir
}

/* Returns the size in kB */
func (r *Resource) SizeInkB() float64 {
	return BaseParser.Round(r.Size / 1024, 1)
}

/* Returns the size in MB */
func (r *Resource) SizeInMB() float64 {
	return BaseParser.Round(r.SizeInkB() / 1024, 1)
}

/*
JS Mime type extraction scrapper: http://webdesign.about.com/od/multimedia/a/mime-types-by-file-extension.htm
var rows = document.body.getElementsByTagName("table")[0].rows,
	tagName = "td",
	ext,
	mime,
	textTypes = [],
	binaryTypes = [];

Object.keys(rows).forEach(function (idx) {
	row = rows[idx];

	if (!row.children || row.children.length == 0 || row.children[0].tagName.toLowerCase() != tagName) {
		return;
	}

	ext = row.children[0].innerText;
	mime = row.children[1].innerText;

	if (mime.indexOf("text") > -1) {
		textTypes.push("\"" + ext + "\"");
	} else {
		binaryTypes.push("\"" + ext + "\"");
	}
})

console.log({
	text: textTypes.join(", "),
	binary: binaryTypes.join(", ")
});
 */
