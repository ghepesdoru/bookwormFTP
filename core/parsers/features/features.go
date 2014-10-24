package features

import (
	BaseParser "github.com/ghepesdoru/bookwormFTP/core/parsers/base"
	Commands "github.com/ghepesdoru/bookwormFTP/core/commands"
	"fmt"
)

const (
	ColonChar = 58
	DefaultSeparator = " "
	EOL = "\r\n"
	EmptyString = ""
)

/* Default error definitions */
var (
	ERR_UnsupportedFeature 	= fmt.Errorf("The specified feature is not supported.")
	ERR_EmptyParameters		= fmt.Errorf("No parameters supported for current feature.")
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
func FromFeaturesList(list []byte) *Features {
	var features *Features = NewFeatures()
	var lines [][]byte
	features.features = make(map[string]string)

	lines = BaseParser.SplitLines(list)
	for _, l := range lines {
		features.extractFeature(l)
	}

	if len(features.features) > 0 {
		features.hasFeatures = true
	}

	return features
}

/* Add a feature */
func (f *Features) AddFeature(feature string, params string) {
	feature = Commands.ToStandardCommand(feature)

	if len(feature) > 0 {
		if Commands.IsValid(feature) && !Commands.IsMandatory(feature) && !f.Supports(feature) {
			f.features[feature] = params

			if f.hasFeatures == false {
				f.hasFeatures = true
			}
		}
	}
}

/* Get the specified feature parameters */
func (f *Features) GetParameters(feature string) (params string, err error) {
	feature = Commands.ToStandardCommand(feature)
	if f.Supports(feature) {
		if params = f.features[feature]; params == EmptyString {
			err = ERR_EmptyParameters
		}
	} else {
		err = ERR_UnsupportedFeature
	}

	return
}

/* Checks if the current Features collection contains any feature */
func (f *Features) HasFeatures() bool {
	return f.hasFeatures
}

/* Removes the specified feature from the current features set */
func (f *Features) RemoveFeature(feature string) {
	if f.Supports(feature) {
		delete(f.features, feature)

		if len(f.features) == 0 {
			f.hasFeatures = false
		}
	}
}

/* Checks if a specified feature exists */
func (f *Features) Supports(feature string) bool {
	if Commands.IsMandatory(feature) {
		return true
	} else if f.HasFeatures() {
		_, ok := f.features[feature]
		return ok
	}

	return false
}

/* Extracts the feature from specified line and adds it to the features list */
func (f *Features) extractFeature(line []byte) {
	var feature, params []byte
	var contentFound, featureFound, hasParam bool
	var start, end int = -1, -1
	var length int = len(line) - 1

	/* Extract the feature and it's params */
	for i, c := range line {
		if !contentFound {
			if !BaseParser.IsWhitespace(c) && c != ColonChar {
				contentFound = true
				start = i
			}
			/* Consume start of string spaces */
		} else {
			if !featureFound {
				if BaseParser.IsWhitespace(c) || c == ColonChar {
					/* Default/alternate separator found */
					end = i
					feature = line[start:end]
					featureFound = true

					/* Mark the starting point for the parameters */
					if i+1 < length {
						hasParam = true
						start = i+1
					} else {
						/* No feature parameters */
						break
					}

					if len(BaseParser.Trim(feature)) == 0 {
						/* Empty feature line! */
						return
					}
				} else if i == length {
					feature = line[start:i + 1]
				}
			} else {
				end = i
			}
		}
	}

	if featureFound && hasParam {
		params = line[start:end+1]
	}

	f.AddFeature(string(feature), string(params))
}

/* To string conversion */
//func (f *Features) String() string {
//	var asString []string = []string{"Features:"}
//
//	for feature, params := range f.features {
//		asString = append(asString, feature + DefaultSeparator + params)
//	}
//
//	if len(asString) > 0 {
//		asString = append(asString, "END")
//	}
//
//	return strings.Join(asString, EOL)
//}
