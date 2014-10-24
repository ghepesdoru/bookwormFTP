package resource

import (
	Access "github.com/ghepesdoru/bookwormFTP/core/access"
	"time"
)

/* Resource type */
type ResourceType int
const(
	TYPE_File ResourceType = iota
	TYPE_Folder
	TYPE_Link
)

type MIMEType int
const (
	MIME_Unknown MIMEType = iota
)

type Charset int
const (
	CHARSET_Unknown Charset = iota
	CHARSET_UTF8
)

var (
	UnknownTime = time.Unix(0, 0)
)

/* File type definition */
type Resource struct {
	Size		int
	Modify		*time.Time
	Create		*time.Time
	Type		ResourceType
	InFocus		bool
	Unique		string
	Permissions	*Access.AccessRights
	Language	string
	MIME		MIMEType
	Charset		Charset
	Content		*[]Resource
}

/* Instantiates a new resource */
func NewResource(size int, modify *time.Time, create *time.Time, rType ResourceType, focus bool, unique string, access *AccessRights.AccessRights, lang string, mime MIMEType, charset Charset) {
	return &Resource{size, modify, create, rType, focus, unique, access, lang, mime, charset}
}

/* Extracts the resource from a MLSx formatted list */
func FromMLSxList(list []byte) (res *Resource, err error) {
	// TODO: Continue from here
}

/* Extracts the resource from a List formatted list */
func FromListList(list []byte) (res *Resource, err error) {
	// TODO: Continue from here
}
