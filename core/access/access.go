package access

var (
	KnownPermissions = map[byte]Perm {
		97: PERM_Append, 	99: PERM_Create, 	100: PERM_Delete, 	101: PERM_Execute,
		102: PERM_Rename, 	108: PERM_List, 	109: PERM_Make, 	112: PERM_Purge,
		114: PERM_Retrievable, 					119: PERM_Storable,
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
func NewAccessRights(perm []Perm) *AccessRights {
	return &AccessRights{perm}
}

/* Instantiates a new empty AccessRights */
func NewEmptyAccessRights() *AccessRights {
	return &AccessRights{[]Perm{}}
}

/* Create a new AccessRights instance from a perm input */
func FromPermString(p []byte) *AccessRights {
	var perm []Perm

	for _, c := range p {
		if v, ok := KnownPermissions[c]; ok {
			perm = append(perm, v)
		}
	}

	return NewAccessRights(perm)
}

/* Checks if the current AccessRights contain the specified permission */
func (a *AccessRights) Contains (perm Perm) bool {
	for _, p := range a.perm {
		if p == perm {
			return true
		}
	}

	return false
}
