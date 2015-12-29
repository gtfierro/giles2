package archiver

import (
	"gopkg.in/mgo.v2/bson"
	"reflect"
	"testing"
)

func BenchmarkDictFromBson1(b *testing.B) {
	test := bson.M{"a": "b", "c": "d"}
	for i := 0; i < b.N; i++ {
		DictFromBson(test)
	}
}

func TestDictFromBson(t *testing.T) {
	for _, test := range []struct {
		in  bson.M
		out Dict
	}{
		{
			bson.M{"a": "b"},
			Dict{"a": "b"},
		},
		{
			bson.M{"a": "b", "c": "d"},
			Dict{"a": "b", "c": "d"},
		},
		{
			bson.M{"a": "b", "c": 3},
			Dict{"a": "b"},
		},
	} {
		got := DictFromBson(test.in)
		if !reflect.DeepEqual(got, test.out) {
			t.Errorf("Dict \n%v\n should be %v but was %v\n", test.in, test.out, got)
		}
	}
}

func TestConvertTime(t *testing.T) {
	for _, test := range []struct {
		time    uint64
		inUnit  UnitOfTime
		result  uint64
		outUnit UnitOfTime
	}{
		{1, UOT_S, 1000, UOT_MS},
		{9, UOT_S, 9000000000, UOT_NS},
		{123456789876, UOT_NS, 123, UOT_S},
		{123456789876, UOT_NS, 123456, UOT_MS},
		{123456789876, UOT_S, 123456789876000000, UOT_US},
	} {
		res, err := convertTime(test.time, test.inUnit, test.outUnit)
		if err != nil {
			t.Error(err)
		}
		if res != test.result {
			t.Errorf("Converting %v %v to %v should be %v but was %v", test.time, test.inUnit, test.outUnit, test.result, res)
		}
	}
}
