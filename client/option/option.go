package option

import (
	"strings"
	"reflect"
)

/* Definition of option type */
type Option struct {
	name string
	vType reflect.Type
	defVal interface {}
	value interface {}
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

/* Checks if the current option has the specified value */
func (o *Option) Is(v interface {}) bool {
	return o.Value() == v
}

/* Option name getter */
func (o *Option) Name() (string) {
	return o.name
}

/* Reset the option to it's default value */
func (o *Option) Reset() {
	o.value = o.defVal
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

/* Option value as string */
func (o *Option) ToString() string {
	if v, ok := o.value.(string); ok {
		return v
	}

	return ""
}

/* Option value getter */
func (o *Option) Value() interface {} {
	return o.value
}
