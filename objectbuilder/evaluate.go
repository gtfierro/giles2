package objectbuilder

import (
	"reflect"
)

func Parse(s string) []Operation {
	l := NewExprLexer(s)
	exParse(l)
	return l.operations
}

type Operation interface {
	Eval(interface{}) interface{}
}

type ArrayOperator struct {
	index       int
	slice       bool
	slice_start int
	slice_end   int
	all         bool
}

func (o *ArrayOperator) Eval(i interface{}) interface{} {
	val := reflect.ValueOf(i)
	kind := val.Type().Kind()
	isarray := (kind == reflect.Slice || kind == reflect.Array)

	if !isarray || o.all {
		return i
	}

	length := val.Len()

	// execute slice
	if o.slice {
		if o.slice_end > length {
			o.slice_end = length
		}
		return val.Slice(o.slice_start, o.slice_end).Interface()
	}

	// return index
	if o.index > length {
		return val.Index(length - 1).Interface()
	} else if o.index < 0 {
		return val.Index(0).Interface()
	}

	return val.Index(o.index).Interface()
}

type ObjectOperator struct {
	key string
}

func (o *ObjectOperator) Eval(i interface{}) interface{} {
	val := reflect.ValueOf(i)
	kind := val.Type().Kind()
	ismap := (kind == reflect.Map)
	isstruct := (kind == reflect.Struct)

	if !(ismap || isstruct) {
		return i
	}

	if ismap {
		return val.MapIndex(reflect.ValueOf(o.key)).Interface()
	}
	// is struct
	return val.FieldByName(o.key).Interface()
}
