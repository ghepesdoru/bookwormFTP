package command

import(
	"testing"
	"fmt"
	Response "github.com/ghepesdoru/bookwormFTP/core/response"
)

var (
	CommandName = "PORT"
	CommandParams = "param=value"
	CommandReplyStatus = []int{200,220}
	ErrorContent = "Test error"
	Error2Content = "Test error 2"
	ErrorTest = fmt.Errorf(ErrorContent)
	Error2Test = fmt.Errorf(Error2Content)
)

func TestCommand(t *testing.T) {
	command := NewCommand(CommandName, CommandParams, CommandReplyStatus)

	if command.Name() != CommandName {
		t.Fatal("Invalid command instance name.", command.Name())
	}

	if !command.HasParameters() {
		t.Fatal("Command not remembering it's parameters.")
	}

	if command.Parameters() != CommandParams {
		t.Fatal("Invalid command parameters.", command.Parameters())
	}

	if command.ExpectedStatus() != "200|220" {
		t.Fatal("Invalid command expected statuses.", command.ExpectedStatus())
	}

	if !command.Success() {
		t.Fatal("Successfull command reporting failure.")
	}

	response := Response.NewResponse(220, []byte("Server ready"), false)
	command.AttachResponse(response, ErrorTest)

	if command.Success() {
		t.Fatal("Failing command reporting success.")
	}

	if command.LastError().Error() != command.Error() {
		t.Fatal("Invalid last error manipulation. Differences between the error and it's .Error() result.")
	}

	if !command.IsValidResponse() {
		t.Fatal("Valid response seen as invalid.")
	}

	if command.Response() != response {
		t.Fatal("Invalid response manipulation.")
	}

	command.FlushErrors()
	if len(command.Errors()) > 0 {
		t.Fatal("Invalid errors flushing.")
	}

	command.AddError(Error2Test)
	if command.Success() {
		t.Fatal("Invalid success status after manually adding a contextual error.")
	}
}
