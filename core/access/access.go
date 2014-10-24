package access

import (
	"strings"
)

var (
	KnownPermissions = map[string]bool {
		"a": PERM_Append, 	"c": PERM_Create, 	"d": PERM_Delete, 	"e": PERM_Execute,
		"f": PERM_Rename, 	"l": PERM_List, 	"m": PERM_Make, 	"p": PERM_Purge,
		"r": PERM_Retrievable, 					"w": PERM_Storable,
	}
)

/* Permission type definition */
type Perm int
const (
	PERM_Append Perm = iota
	PERM_Create
	PERM_Delete
	PERM_Execute
	PERM_Rename
	PERM_List
	PERM_Make
	PERM_Purge
	PERM_Retrievable
	PERM_Storable
)

/* Access rights type definition */
type AccessRights struct {
	perm []Perm
}

/* Given an array of permissions instantiates a new AccessRights */
func NewAccessRights(perm []Perm) {
	return &AccessRights{perm}
}

/* Create a new AccessRights instance from a perm input */
func FromPermString(perm string) {
	var perm []Perm

	for identifier, v := range KnownPermissions {
		if strings.Contains(identifier) {
			perm = append(perm, v)
		}
	}

	return NewAccessRights(perm)
}
