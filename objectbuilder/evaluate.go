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
	isarray := val.Type().Kind() == reflect.Slice

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
