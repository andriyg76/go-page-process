package processor

import (
	"encoding/json"
	"reflect"

	"github.com/aymerick/raymond"
)

func registerSpecHelpers(tpl *raymond.Template) {
	tpl.RegisterHelper("if", compatIfHelper)
	tpl.RegisterHelper("unless", compatUnlessHelper)
}

func compatIfHelper(conditional interface{}, options *raymond.Options) interface{} {
	if includeZero(options) || raymond.IsTrue(conditional) {
		return options.Fn()
	}
	return options.Inverse()
}

func compatUnlessHelper(conditional interface{}, options *raymond.Options) interface{} {
	if includeZero(options) || raymond.IsTrue(conditional) {
		return options.Inverse()
	}
	return options.Fn()
}

func includeZero(options *raymond.Options) bool {
	include, ok := options.HashProp("includeZero").(bool)
	if !ok || !include {
		return false
	}
	return isNumericZero(options.Param(0))
}

func isNumericZero(value interface{}) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i == 0
		}
		if f, err := v.Float64(); err == nil {
			return f == 0
		}
		return false
	}

	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return false
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return rv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0
	default:
		return false
	}
}
