package command

import (
	"strings"
	"strconv"
	Response "github.com/ghepesdoru/bookwormFTP/core/response"
	Commands "github.com/ghepesdoru/bookwormFTP/core/commands"
)

/* Client Command type definition */
type Command struct {
	command 			string
	parameters 			string
	status				map[int]bool	/* List of accepted statuses for success case */
	ok					bool
	err					[]error
	response			*Response.Response
}

/* Client Command builder */
func NewCommand(command string, parameters string, statuses []int) *Command {
	command = Commands.ToStandardCommand(command)
	status := make(map[int]bool)

	for _, s := range statuses {
		status[s] = true
	}

	return &Command{command, parameters, status, true, []error{}, &Response.Response{}}
}

/* Command name getter */
func (c *Command) Name() string {
	return c.command
}

/* Checks if the specified status can represent a command success status */
func (c *Command) IsExpectedStatus (status int) bool {
	if _, ok := c.status[status]; ok {
		return ok
	}

	return false
}

/* Generates a list of expected statuses separated by pipes */
func (c *Command) ExpectedStatus() string {
	var statuses []string

	for s, _ := range c.status {
		statuses = append(statuses, strconv.Itoa(s))
	}

	return strings.Join(statuses, "|")
}

/* Checks if the current command completed successfully after it's execution */
func (c *Command) Success() bool {
	return c.ok
}

/* Grabs the current command errors */
func (c *Command) Errors() []error {
	return c.err
}

/* Returns the first error message found */
func (c *Command) Error() string {
	if len(c.err) > 0 {
		return c.err[0].Error()
	}

	return ""
}

/* Flushes all errors */
func (c *Command) FlushErrors() []error {
	err := c.err
	c.err = []error{}
	return err
}

/* Grabs the last error triggered by the current command */
func (c *Command) LastError() error {
	if l := len(c.err); l > 0 {
		return c.err[l - 1]
	}

	return nil
}

/* Command parameters getter */
func (c *Command) Parameters() string {
	return c.parameters
}

/* Checks if the current command has attached parameters */
func (c *Command) HasParameters() bool {
	return len(c.parameters) > 0
}

/* Adds a new error to the command errors list */
func (c *Command) AddError(err error) {
	if err != nil {
		/* Invalidate command */
		c.err = append(c.err, err)
		c.ok = false
	}
}

/* Marks the command as being executed, adding it's response */
func (c *Command) AttachResponse(response *Response.Response) {
	c.response = response
}

/* Checks if an acctual response was attached to the current command */
func (c *Command) ValidResponse() bool {
	return c.response != nil && c.response.Status() != 0
}

/* Command response getter */
func (c *Command) Response() *Response.Response {
	return c.response
}
