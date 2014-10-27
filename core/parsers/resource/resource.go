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
)

var (
	ExtractListParts = regexp.MustCompilePOSIX(`(^[ldrwx-]*)(?: *)([[:digit:]]*)(?: *)([[:alpha:]]*)(?: *)([[:alpha:]]*)(?: *)([0-9]*)(?: *)([A-Za-z]*.[0-9]*(?: *)[0-9]*)(?: *)([A-Za-z*_.-]+)*(?:.*)`)
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
func FromListList(list []byte) (res *Resource, err error) {
	fmt.Println(ExtractListParts.FindAll(list, -1))
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

/* Checks if the current resource is the current container directory */
func (r *Resource) IsCurrentDir() bool {
	return r.Type == TYPE_CDir
}

func (r *Resource) IsParentDir() bool {
	return r.Type == TYPE_PDir
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
