package settings

import(
	"testing"
)

const (
	INIT_OPT = "initial option"
	ADDED_OPT = "added option"
)

func Test(t *testing.T) {
	settings := NewSettings(NewOption(INIT_OPT, false))
	settings.Add(ADDED_OPT, true)

	if !settings.Has(INIT_OPT) {
		t.Fatal("Invalid settings getter for initialization option.")
	}

	if o := settings.Get(ADDED_OPT); o.Name() != ADDED_OPT {
		t.Fatal("Invalid settings getter for added option.")
	}

	settings.Remove(INIT_OPT)
	if settings.Has(INIT_OPT) {
		t.Fatal("The initialization option was not removed.")
	}

	settings.Remove(ADDED_OPT)
	if settings.Has(ADDED_OPT) {
		t.Fatal("The added option was not removed.")
	}
}
