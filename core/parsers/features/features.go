package features

import (
	"strings"
	"fmt"
)

const (
	EmptyString = ""
	DefaultSeparator = " "
	AlternativeSeparator = ":"
	WindowsEOL = "\r\n"
	NixEOL = "\n"
)

/* Features type */
type Features struct{
	features map[string]string
	hasFeatures bool
}

/* Build a new features instance */
func NewFeatures() *Features {
	return &Features{make(map[string]string), false}
}

/* Given a FEAT reply, generates a new Features instance with parsed data. */
func FromFeaturesList(list string) *Features {
	var features *Features = NewFeatures()
	var parts []string
	var feature, params string

	if parts = strings.Split(list, WindowsEOL); len(parts) == 0 {
		parts = strings.Split(list, NixEOL)
	}

	for _, line := range parts {
		line = strings.ToUpper(strings.TrimSpace(line))
		aux := strings.Split(line, DefaultSeparator)
		feature = EmptyString
		params = EmptyString

		if l := len(aux); l > 0 {
			if strings.Contains(line, AlternativeSeparator) {
				aux2 := strings.Split(line, AlternativeSeparator)

				if len(aux2) > 1 {
					params = strings.TrimSpace(strings.Join(aux2[1:], DefaultSeparator))
				}

				feature = aux2[0]
			} else {
				if l > 1 {
					params = strings.Join(aux[1:], DefaultSeparator)
				}

				feature = aux[0]
			}
		}

		switch feature {
		case "EPRT", "EPSV", "MDTM", "SIZE", "TVFS", "UTF8", "HOST", "LANG":
			features.features[feature] = params
		case "AUTH":
			features.features[feature] = params
			fmt.Println("Auth feature!")
		case "MLSD", "MLST":
			features.features[feature] = params
			fmt.Println("Parser pattern here.")
		}
	}

	if len(features.features) > 0 {
		features.hasFeatures = true
	}

	return features
}

/* Checks if the current Features collection contains any feature */
func (f *Features) HasFeatures() bool {
	return f.hasFeatures
}

/* To string conversion */
func (f *Features) String() string {
	var asString []string = []string{"Features:"}

	for feature, params := range f.features {
		asString = append(asString, feature + DefaultSeparator + params)
	}

	if len(asString) > 0 {
		asString = append(asString, "END")
	}

	return strings.Join(asString, WindowsEOL)
}
