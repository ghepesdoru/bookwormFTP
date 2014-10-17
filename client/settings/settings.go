package settings

import (
	"strings"
	Option "github.com/ghepesdoru/bookwormFTP/client/option"
)

/* ClientSettings type definition */
type Settings struct {
	content map[string]*Option.Option
}

/* Settings builder */
func NewSettings(options ...*Option.Option) (settings *Settings) {
	settings = &Settings{make(map[string]*Option.Option)}

	for _, o := range options {
		if len(strings.TrimSpace(o.Name())) > 0 {
			settings.content[o.Name()] = o
		}
	}

	return
}

/* Option builder wrapper for ease of usage */
func NewOption(name string, defaultValue interface{}) *Option.Option {
	return Option.NewOption(name, defaultValue)
}

/* Add a new option to the Settings object, or return the existing option with the specified name */
func (s *Settings) Add(optionName string, defaultValue interface {}) *Option.Option {
	if !s.Has(optionName) {
		s.content[optionName] = NewOption(optionName, defaultValue)
	}

	return s.Get(optionName)
}

/* Option getter */
func (s *Settings) Get(optionName string) *Option.Option {
	if o, ok := s.content[optionName]; ok {
		return o
	}

	return &Option.Option{}
}

/* Checks if the current Settings object contains an option with the specified name */
func (s *Settings) Has(optionName string) bool {
	if _, ok := s.content[optionName]; ok {
		return true
	}

	return false
}

/* Option delete functionality */
func (s *Settings) Remove(optionName string) bool {
	if _, ok := s.content[optionName]; ok {
		delete(s.content, optionName)
	}

	return s.content[optionName] == nil
}
