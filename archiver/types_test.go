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
		if !reflect.DeepEqual(*got, test.out) {
			t.Errorf("Dict \n%v\n should be %v but was %v\n", test.in, test.out, got)
		}
	}
}
