package objectbuilder

import (
	"fmt"
	"reflect"
)

func Parse(s string) []Operation {
	l := NewExprLexer(s)
	exParse(l)
	if l.error != nil {
		fmt.Println(l.error)
	}
	return l.operations
}

func Eval(ops []Operation, v interface{}) interface{} {
	for _, op := range ops {
		v = op.Eval(v)
	}
	return v
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

func (o ArrayOperator) Eval(i interface{}) interface{} {
	val := reflect.ValueOf(i)
	if !val.IsValid() {
		return nil
	}
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
		return val.Index(length + o.index).Interface()
	}

	ret := val.Index(o.index)
	if !ret.IsValid() {
		return nil
	}

	return ret.Interface()
}

type ObjectOperator struct {
	key string
}

func (o ObjectOperator) Eval(i interface{}) interface{} {
	val := reflect.ValueOf(i)
	if !val.IsValid() {
		return nil
	}
	kind := val.Type().Kind()
	if kind == reflect.Ptr {
		val = val.Elem()
		if !val.IsValid() {
			return nil
		}
		kind = val.Type().Kind()
	}
	ismap := (kind == reflect.Map)
	isstruct := (kind == reflect.Struct)

	if !(ismap || isstruct) {
		return i
	}

	if ismap {
		try := val.MapIndex(reflect.ValueOf(o.key))
		if try.IsValid() {
			return try.Interface()
		} else {
			return i
		}
	}
	// is struct
	return val.FieldByName(o.key).Interface()
}
