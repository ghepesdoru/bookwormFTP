package credentials

import (
	"fmt"
	"strings"
)

const EMPTY_SPACE = ' '

/* Bookworm client user data */
type Credentials struct {
	username string
	password string
}

/* Define default errors */
var (
	ERR_UsernameToShort = fmt.Errorf("Invalid username. User name to short")
)

/* Credentials structure builder */
func NewCredentials(username string, password string) (credentials *Credentials, err error) {
	var u []string
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	for _, c := range username {
		if c != EMPTY_SPACE {
			u = append(u, string(c))
		}
	}

	username = strings.Join(u, "")

	if len(username) == 0 {
		err = ERR_UsernameToShort
	}

	if err == nil {
		credentials = &Credentials{username, password}
	}

	return
}

/* Credentials username getter */
func (c *Credentials) Username() (string) {
	return c.username
}

/* Credentials user password getter */
func (c *Credentials) Password() (string) {
	return c.password
}
