package settings

import (
	"strings"
	"reflect"
)

/* Define the Setting type */
type Option struct {
	name string
	vType reflect.Type
	defVal interface {}
	value interface {}
}

/* ClientSettings type definition */
type Settings struct {
	content map[string]*Option
}

/* Settings builder */
func NewSettings(options ...*Option) (settings *Settings) {
	settings = &Settings{make(map[string]*Option)}

	for _, o := range options {
		if len(strings.TrimSpace(o.Name())) > 0 {
			settings.content[o.Name()] = o
		}
	}

	return
}

/* Option getter */
func (s *Settings) Get(optionName string) *Option {
	if o, ok := s.content[optionName]; ok {
		return o
	}

	return &Option{}
}

/* Checks if the current Settings object contains an option with the specified name */
func (s *Settings) Has(optionName string) bool {
	if _, ok := s.content[optionName]; ok {
		return true
	}

	return false
}

/* Add a new option to the Settings object, or return the existing option with the specified name */
func (s *Settings) Add(optionName string, defaultValue interface {}) *Option {
	if !s.Has(optionName) {
		s.content[optionName] = NewOption(optionName, defaultValue)
	}

	return s.Get(optionName)
}

/* Option delete functionality */
func (s *Settings) Remove(optionName string) bool {
	if _, ok := s.content[optionName]; ok {
		delete(s.content, optionName)
	}

	return s.content[optionName] == nil
}

/* Option builder */
func NewOption(name string, defaultValue interface{}) (opt *Option) {
	name = strings.TrimSpace(name)
	if len(name) == 0 {
		/* Invalid option name */
		return
	} else {
		opt = &Option{name, reflect.TypeOf(defaultValue), defaultValue, defaultValue}
	}

	return
}

/* Option name getter */
func (o *Option) Name() (string) {
	return o.name
}

/* Option value getter */
func (o *Option) Value() interface {} {
	return o.value
}

/* Option value as string */
func (o *Option) ToString() string {
	if v, ok := o.value.(string); ok {
		return v
	}

	return ""
}

/* Option value as boolean */
func (o *Option) ToBool() bool {
	if v, ok := o.value.(bool); ok {
		return v
	}

	return false
}

/* Option value as int */
func (o *Option) ToInt() int {
	if v, ok := o.value.(int); ok {
		return v
	}

	return 0
}

/* Checks if the current option has the specified value */
func (o *Option) Is(v interface {}) bool {
	return o.Value() == v
}

/* Set a new value for the current option */
func (o *Option) Set(newValue interface {}) (err error) {
	if reflect.TypeOf(newValue) != o.vType {
		/* Invalid value for specified type */
	} else {
		o.value = newValue
	}

	return
}

/* Reset the option to it's default value */
func (o *Option) Reset() {
	o.value = o.defVal
}
