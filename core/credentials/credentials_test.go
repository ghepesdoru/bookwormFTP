package credentials

import "testing"

const(
	WRONG_Username = ""
	SPACED_Username = "   aA a "
	SPACED_Password = "  ss  s  "
	CORRECT_Username = "aAa"
	CORRECT_Password = "ss  s"
)


func TestCredentials(t *testing.T) {
	credentials, err := NewCredentials(WRONG_Username, CORRECT_Password)
	if err != ERR_UsernameToShort {
		t.Fatal("Invalid credentials sanitizing. Username to short.")
	}

	credentials, err = NewCredentials(SPACED_Username, SPACED_Password)
	if err != nil {
		t.Fatal("Invalid credentials initialization process. Failed on valid data.")
	}

	if credentials.Username() != CORRECT_Username {
		t.Fatal("Invalid credentials username sanitizing.")
	}

	if credentials.Password() != CORRECT_Password {
		t.Fatal("Invalid credentials password sanitizing.")
	}
}
