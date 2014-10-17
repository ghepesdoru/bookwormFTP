package option

import "testing"

type OptionTest struct{
	name string
	defVal interface{}
	setTo interface{}
	asBool bool
	asInt int
	asString string
}

var (
	Options = []OptionTest{
		OptionTest{
			name: "BoolOption",
			defVal: false,
			setTo: true,
			asBool: true,
			asInt: 0,
			asString: "",
		},
		OptionTest{
			name: "IntOption",
			defVal: 1,
			setTo: 2,
			asBool: false,
			asInt: 2,
			asString: "",
		},
		OptionTest{
			name: "StringOption",
			defVal: "some string",
			setTo: "some other string",
			asBool: false,
			asInt: 0,
			asString: "some other string",
		},
	}
)

/* Execute this generic test on specified option and compare with the specified reference */
func test(opt *Option, ref OptionTest, t *testing.T) {
	if opt.Name() != ref.name {
		t.Fatal("Invalid option name.", opt.Name())
	}

	if !opt.Is(ref.defVal) {
		t.Fatal("Invalid option default value", opt.Value())
	}

	if opt.Value() != ref.defVal {
		t.Fatal("Invalid option value getter", opt.Value())
	}

	opt.Set(ref.setTo)
	if !opt.Is(ref.setTo) {
		t.Fatal("Invalid option value change.", opt.Value())
	}

	if opt.ToBool() != ref.asBool {
		t.Fatal("Invalid option conversion to boolean.")
	}

	if opt.ToInt() != ref.asInt {
		t.Fatal("Invalid option conversion to int.")
	}

	if opt.ToString() != ref.asString {
		t.Fatal("Invalid option conversion to string.")
	}

	opt.Reset()
	if opt.Value() != ref.defVal {
		t.Fatal("Invalid option value reset.")
	}
}

func TestBool(t *testing.T) {
	testCase := Options[0]
	test(NewOption(testCase.name, testCase.defVal), testCase, t)
}

func TestInt(t *testing.T) {
	testCase := Options[1]
	test(NewOption(testCase.name, testCase.defVal), testCase, t)
}

func TestString(t *testing.T) {
	testCase := Options[2]
	test(NewOption(testCase.name, testCase.defVal), testCase, t)
}
